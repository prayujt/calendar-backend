package main

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

type Calendar struct {
	Id        string `json:"id" database:"id"`
	Name      string `json:"name" database:"name"`
	Color     string `json:"color" database:"color"`
	IsDefault bool   `json:"isDefault" database:"is_default"`
}

// GET /calendars
func getCalendars(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	calendars := []Calendar{}
	Query(&calendars,
		`
		SELECT calendars.id AS id, calendars.name AS name, calendars.color AS color, calendars.is_default AS isDefault
		FROM calendar_members
		JOIN calendars ON calendars.id = calendar_members.calendar_id
		WHERE user_id = $1
		`,
		session.Identity.Id,
	)

	if len(calendars) == 0 {
		calendars = []Calendar{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendars)
}

// POST /calendars
func createCalendar(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	var calendar Calendar
	err := json.NewDecoder(r.Body).Decode(&calendar)
	if err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	calendar.Id = uuid.New().String()
	if calendar.IsDefault {
		calendar.Color = "#93c4fd"
	}

	_, err = Execute(
		`
		INSERT INTO calendars (id, name, color, is_default)
		VALUES ($1, $2, $3, $4)
		`,
		calendar.Id,
		calendar.Name,
		calendar.Color,
		calendar.IsDefault,
	)
	if err != nil {
		http.Error(w, `{"error": "Error creating calendar"}`, http.StatusInternalServerError)
		return
	}

	_, err = Execute(
		`
		INSERT INTO calendar_members (calendar_id, user_id)
		VALUES ($1, $2)
		`,
		calendar.Id,
		session.Identity.Id,
	)
	if err != nil {
		http.Error(w, `{"error": "Error creating calendar"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendar)
}

// PUT /calendars/{id}
func updateCalendar(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	calendarId := vars["id"]

	var calendar Calendar
	err := json.NewDecoder(r.Body).Decode(&calendar)
	if err != nil {
		http.Error(w, `{"error": "Invalid JSON"}`, http.StatusBadRequest)
		return
	}

	_, err = Execute(
		`
		UPDATE calendars
		SET name = $1, color = $2
		WHERE id = $3
		`,
		calendar.Name,
		calendar.Color,
		calendarId,
	)
	if err != nil {
		http.Error(w, `{"error": "Error updating calendar"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(calendar)
}

// DELETE /calendars/{id}
func deleteCalendar(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	vars := mux.Vars(r)
	calendarId := vars["id"]

	_, err := Execute(
		`
		DELETE FROM calendars
		WHERE id = $1
		`,
		calendarId,
	)
	if err != nil {
		http.Error(w, `{"error": "Error deleting calendar"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
