package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Event struct {
	Id          string  `json:"id" database:"id"`
	UserId      string  `json:"userId" database:"user_id"`
	Title       string  `json:"title" database:"title`
	Description *string `json:"description" database:"description"`
	Duration    int     `json:"duration" database:"duration"`
	Date        string  `json:"date" database:"date`
	Accepted    bool    `json:"accepted" database:"accepted`
}

type GenerateEventRequest struct {
	Content string `json:"content"`
}

// GET /events
func getEvents(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	userId := session.Identity.Id

	var events []Event
	Query(&events, "SELECT * FROM events WHERE user_id = $1", userId)

	if len(events) == 0 {
		events = []Event{}
	}
	json.NewEncoder(w).Encode(events)
}

// POST /events
func postEvent(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var event Event
	if err := json.NewDecoder(r.Body).Decode(&event); err != nil {
		http.Error(w, `{"error": "Invalid request"}`, http.StatusBadRequest)
		return
	}

	event.UserId = session.Identity.Id

	_, err := Execute(
		"INSERT INTO events (id, user_id, title, description, duration, date, accepted) VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)",
		event.UserId, event.Title, event.Description, event.Duration, event.Date, event.Accepted,
	)
	if err != nil {
		log.Println(err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(event)
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

	// Define the OpenAI function schema
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
				},
				"required": []string{"title", "duration", "date"},
			},
		},
	}

	response, err := client.CreateChatCompletion(
		context.Background(),
		openai.ChatCompletionRequest{
			Model: openai.GPT4o,
			Messages: []openai.ChatCompletionMessage{
				{
					Role: openai.ChatMessageRoleSystem,
					Content: fmt.Sprintf(`
						You are an event information parser.
						Extract the event title, description, duration (in minutes), and date from the following text.
						Make sure the date is in ISO 8601 format. If the date is something like "today" or "tomorrow", or "next tuesday", convert it to the appropriate date.
						For context, the exact time right now is %s (in ISO 8601 format).
						By default, if the duration is not specified, it should be 60 minutes.
					`, time.Now().Format(time.RFC3339)),
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

	var functionResponse struct {
		Title       string    `json:"title"`
		Description string    `json:"description"`
		Duration    int       `json:"duration"`
		Date        time.Time `json:"date"`
	}

	err = json.Unmarshal([]byte(response.Choices[0].Message.FunctionCall.Arguments), &functionResponse)
	if err != nil {
		log.Println("Error parsing function call response:", err)
		http.Error(w, `{"error": "Invalid event data format"}`, http.StatusInternalServerError)
		return
	}

	_, err = Execute(
		`
		INSERT INTO events (id, user_id, title, description, duration, date, accepted)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, $6)
		`,
		session.Identity.Id, functionResponse.Title, functionResponse.Description, functionResponse.Duration, functionResponse.Date.Format(time.RFC3339), true,
	)
	if err != nil {
		log.Println("Error inserting event into database:", err)
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(functionResponse)
}
