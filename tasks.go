package main

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
)

type Task struct {
	Id          string  `json:"id" database:"id"`
	UserId      string  `json:"userId" database:"user_id"`
	CalendarId  string  `json:"calendarId" database:"calendar_id"`
	Title       string  `json:"title" database:"title`
	Description *string `json:"description" database:"description"`
	Duration    int     `json:"duration" database:"duration"`
	Deadline    string  `json:"deadline" database:"deadline"`
	Difficulty  int     `json:"difficulty" database:"difficulty"`
	Priority    int     `json:"priority" database:"priority"`
	Completed   bool    `json:"completed" database:"completed"`
}

// GET /tasks
func getTasks(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	userId := session.Identity.Id

	var tasks []Task
	Query(&tasks,
		`
		SELECT * FROM tasks
		WHERE user_id = $1
		`,
		userId,
	)

	if len(tasks) == 0 {
		tasks = []Task{}
	}
	json.NewEncoder(w).Encode(tasks)
}

// POST /tasks
func createTask(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	userId := session.Identity.Id

	var task Task
	json.NewDecoder(r.Body).Decode(&task)

	_, err := Execute(
		`
		INSERT INTO tasks (user_id, calendar_id, title, description, duration, deadline, difficulty, priority)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`,
		userId,
		task.CalendarId,
		task.Title,
		task.Description,
		task.Duration,
		task.Deadline,
		task.Difficulty,
		task.Priority,
	)

	if err != nil {
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(task)
}

// PUT /tasks/:id
func updateTask(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	userId := session.Identity.Id
	vars := mux.Vars(r)
	taskId := vars["id"]

	var task Task
	json.NewDecoder(r.Body).Decode(&task)

	_, err := Execute(
		`
		UPDATE tasks
		SET calendar_id = $2, title = $3, description = $4, duration = $5, deadline = $6, difficulty = $7, priority = $8, completed = $9
		WHERE id = $1 AND user_id = $10
		`,
		taskId,
		task.CalendarId,
		task.Title,
		task.Description,
		task.Duration,
		task.Deadline,
		task.Difficulty,
		task.Priority,
		task.Completed,
		userId,
	)

	if err != nil {
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(task)
}

// DELETE /tasks/:id
func deleteTask(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		http.Error(w, `{"error": "Unauthorized"}`, http.StatusUnauthorized)
		return
	}

	userId := session.Identity.Id
	vars := mux.Vars(r)
	taskId := vars["id"]

	_, err := Execute(
		`
		DELETE FROM tasks
		WHERE id = $1 AND user_id = $2
		`,
		taskId,
		userId,
	)

	if err != nil {
		http.Error(w, `{"error": "Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"id": taskId})
}
