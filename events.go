package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/sashabaranov/go-openai"
)

type Event struct {
	Id           string  `json:"id" database:"id"`
	CalendarId   string  `json:"calendarId" database:"calendar_id"`
	Title        string  `json:"title" database:"title`
	Description  *string `json:"description" database:"description"`
	Duration     int     `json:"duration" database:"duration"`
	Date         string  `json:"date" database:"date`
	RecurrenceId string  `json:"recurrenceId" database:"recurrence_id"`
}

type GenerateEventRequest struct {
	Content    string `json:"content"`
	CalendarId string `json:"calendarId"`
}

type CreateEventRequest struct {
	CalendarId  string    `json:"calendarId"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Duration    int       `json:"duration"`
	Date        time.Time `json:"date"`
	Recurring   bool      `json:"recurring"`
}

var dateFormat = "2006-01-02T15:04:05Z07:00"

// GET /events
func getEvents(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	userId := session.Identity.Id

	var events []Event
	Query(&events,
		`
		SELECT * FROM events
		WHERE calendar_id IN (SELECT calendar_id FROM calendar_members WHERE user_id = $1)
		`,
		userId,
	)

	if len(events) == 0 {
		events = []Event{}
	}
	json.NewEncoder(w).Encode(events)
}

// POST /events
func createEvent(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var event CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}

	if event.Recurring {
		var newEvent Event
		recurrence_id := uuid.New().String()
		for i := 0; i < 100; i++ {
			event_id := uuid.New().String()
			_, err := Execute(
				`
				INSERT INTO events (id, calendar_id, title, description, duration, date, recurrence_id)
				VALUES ($1, $2, $3, $4, $5, $6, $7)
				`,
				event_id, event.CalendarId, event.Title, event.Description, event.Duration, event.Date.AddDate(0, 0, i*7).Format(time.RFC3339), recurrence_id,
			)
			if err != nil {
				log.Println("Error inserting event into database:", err)
				http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
				return
			}

			if i == 0 {
				newEvent = Event{
					Id:           event_id,
					CalendarId:   event.CalendarId,
					Title:        event.Title,
					Description:  &event.Description,
					Duration:     event.Duration,
					Date:         event.Date.Format(time.RFC3339),
					RecurrenceId: recurrence_id,
				}
			}
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newEvent)
	} else {
		event_id := uuid.New().String()
		_, err := Execute(
			`
			INSERT INTO events (id, calendar_id, title, description, duration, date)
			VALUES ($1, $2, $3, $4, $5, $6)
			`,
			event_id, event.CalendarId, event.Title, event.Description, event.Duration, event.Date.Format(time.RFC3339),
		)
		if err != nil {
			log.Println("Error inserting event into database:", err)
			http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
			return
		}

		newEvent := Event{
			Id:          event_id,
			CalendarId:  event.CalendarId,
			Title:       event.Title,
			Description: &event.Description,
			Duration:    event.Duration,
			Date:        event.Date.Format(time.RFC3339),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(newEvent)
	}
}

// GET /events/{id}
func getEvent(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)

	userId := session.Identity.Id
	eventId := vars["id"]

	var event []Event
	Query(&event, "SELECT * FROM events WHERE id = $1 AND user_id = $2", eventId, userId)

	if len(event) == 0 {
		http.Error(w, `{"error": "Event not found"}`, http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(event[0])
}

// PUT /events/{id}
func updateEvent(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	eventId := vars["id"]
	recurring := r.URL.Query().Get("recurring")

	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}

	if recurring == "true" {
		var events []Event
		Query(&events, "SELECT * FROM events WHERE id = $1", eventId)
		recurrence_id := events[0].RecurrenceId
		oldDate := events[0].Date

		if recurrence_id == "" {
			log.Println("Event is not recurring")
			http.Error(w, `{"error": "Event is not recurring"}`, http.StatusBadRequest)
			return
		}

		oldEventDate, err := time.Parse(dateFormat, oldDate)
		if err != nil {
			log.Println("Error parsing old event date:", err)
			http.Error(w, `{"error": "Invalid event date"}`, http.StatusInternalServerError)
			return
		}

		newEventDate, err := time.Parse(dateFormat, event.Date)
		if err != nil {
			log.Println("Error parsing new event date:", err)
			http.Error(w, `{"error": "Invalid new event date"}`, http.StatusInternalServerError)
			return
		}

		dateDifference := newEventDate.Sub(oldEventDate)
		interval := dateDifference.Seconds() / 86400

		_, err = Execute(
			`
			UPDATE events
			SET title = $1, description = $2, duration = $3, date = date + $6 * INTERVAL '1 day'
			WHERE recurrence_id = $4 AND date >= $5
			`, event.Title, event.Description, event.Duration, recurrence_id, oldDate, interval,
		)
		if err != nil {
			log.Println("Error updating events:", err)
			http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
			return
		}
	} else {
		_, err := Execute(
			`UPDATE events SET title = $1, description = $2, duration = $3, date = $4, recurrence_id = NULL WHERE id = $5`,
			event.Title, event.Description, event.Duration, event.Date, eventId,
		)
		if err != nil {
			log.Println(err)
			http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// DELETE /events/{id}
func deleteEvent(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}
	vars := mux.Vars(r)
	eventId := vars["id"]
	recurring := r.URL.Query().Get("recurring")

	var event []Event
	Query(&event, "SELECT * FROM events WHERE id = $1", eventId)

	if event[0].RecurrenceId != "" && recurring == "true" {
		_, err := Execute("DELETE FROM events WHERE recurrence_id = $1 AND date >= $2", event[0].RecurrenceId, event[0].Date)
		if err != nil {
			log.Println(err)
			http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
			return
		}
	} else {
		_, err := Execute("DELETE FROM events WHERE id = $1", eventId)
		if err != nil {
			log.Println(err)
			http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusOK)
}

// POST /events/generate
func generateEventInformation(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	client := openai.NewClient(os.Getenv("OPENAI_API_KEY"))
	var request GenerateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}

	functions := []openai.FunctionDefinition{
		{
			Name:        "extract_event_details",
			Description: "Extracts the event title, description, duration, and date from the given text",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"title": map[string]string{
						"type":        "string",
						"description": "The title of the event",
					},
					"description": map[string]string{
						"type":        "string",
						"description": "The description of the event",
					},
					"duration": map[string]string{
						"type":        "integer",
						"description": "The duration of the event in minutes",
					},
					"date": map[string]string{
						"type":        "string",
						"description": "The date of the event in ISO 8601 format",
					},
					"recurring": map[string]string{
						"type":        "boolean",
						"description": "Whether the event is recurring or not",
					},
				},
				"required": []string{"title", "duration", "date", "recurring"},
			},
		},
	}

	response, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4oMini,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf(`
						You are an event information parser.
						Extract the event title, description, duration (in minutes), and date from the following text.
						Make sure the date is in ISO 8601 format. If the date is something like "today" or "tomorrow", or "next tuesday", convert it to the appropriate date.
						For context, the exact time right now is %s (in ISO 8601 format and UTC).
						The event details that you will be given below will be in Eastern Time (ET).
						e.g. If you are asked about an event at 5:10 PM, you should convert that to 9:10 PM UTC if it is currently in Eastern Daylight Time (EDT), or 10:10 PM UTC if it is currently EST.
						Similarly, the day provided should be converted to the appropriate date in UTC.
						e.g. If you are given an event at 11:50 PM on the 31st of October, you should convert that to 3:50 AM UTC on the 1st of November.
						Another example is, if given an event at 8:00PM on the 1st of November, you should convert that to 12:00AM UTC on the 2nd of November if it is currently in EDT
						Generate the ISO 8601 date and time for the event in UTC please, taking into account Daylight Saving Time to determine if Eastern Time is currently in EDT or EST.
						By default, if the duration is not specified, it should be 60 minutes.
						For title and description, don't simply extract it word for word. Instead, generate a title and description that captures the essence of the event.
						Ensure the format of the title is in title case, with words capitalized except for articles, prepositions, and conjunctions.
						For the description, if information is given then provide a short description of the event. If no information is given, then leave it blank.
						For example, if the content given is "Meeting John at 5:00 PM", the title could be "Meeting with John" and the description would be blank.
						If the content given is "Meeting John at 5:00 PM to discuss the project", the title could be "Project Discussion with John" and the description could be "Discuss the project with John".
						Additionally, mark if the event is going to be a recurring event or not.
						The date should be the date of the event in the current week, regardless of the current day.
						For example, if the content given is "Meeting John at 5:00 PM every Monday", the title would be "Meeting with John" and the event would be marked as recurring. Also, in this example, the date should be the Monday of the current week, regardless of the current day.
						Again, as a reminder, the exact time right now is %s (in ISO 8601 format and UTC).
					`, time.Now().UTC().Format(dateFormat), time.Now().UTC().Format(dateFormat)),
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: request.Content,
				},
			},
			Functions: functions,
			FunctionCall: openai.FunctionCall{
				Name: "extract_event_details",
			},
		},
	)

	if err != nil {
		log.Println("Error with OpenAI API:", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	var functionResponse CreateEventRequest
	err = json.Unmarshal([]byte(response.Choices[0].Message.FunctionCall.Arguments), &functionResponse)
	if err != nil {
		log.Println("Error parsing function call response:", err)
		http.Error(w, `{"error": "Invalid event data format"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(functionResponse)
}
