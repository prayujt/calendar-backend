package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type Session struct {
	Id       string   `json:"id"`
	Active   bool     `json:"active"`
	Identity Identity `json:"identity"`
}

type Identity struct {
	Id     string `json:"id"`
	State  string `json:"state"`
	Traits Traits `json:"traits"`
}

type Traits struct {
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Username  string `json:"username"`
	Avatar    string `json:"avatar"`
}

var kratosPublicUrl string
var kratosAdminUrl string
var environment string
var mailPassword string

func main() {
	kratosPublicUrl = os.Getenv("KRATOS_PUBLIC_URL")
	if kratosPublicUrl == "" {
		kratosPublicUrl = "https://idp.prayujt.com"
	}
	kratosAdminUrl = os.Getenv("KRATOS_ADMIN_URL")
	if kratosAdminUrl == "" {
		kratosAdminUrl = "http://kratos-admin.kratos.svc.cluster.local"
	}
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		log.Fatal("DATABASE_URL must be set")
	}
	mailPassword = os.Getenv("MAIL_PASSWORD")
	if mailPassword == "" {
		log.Fatal("MAIL_PASSWORD must be set")
	}

	environment = os.Getenv("ENVIRONMENT")
	if environment == "" {
		environment = "development"
	}

	InitDatabase(databaseUrl)
	log.Println("Connected to database")

	if environment == "development" {
		log.Println("Running in development mode")
	} else {
		log.Println("Running in production mode")
		log.Printf("Using Kratos public at: %s", kratosPublicUrl)
		log.Printf("Using Kratos admin at: %s", kratosAdminUrl)
	}

	r := mux.NewRouter()

	r.HandleFunc("/events", getEvents).Methods("GET")
	r.HandleFunc("/events/{id}", getEvent).Methods("GET")
	r.HandleFunc("/events", createEvent).Methods("POST")
	r.HandleFunc("/events/generate", generateEventInformation).Methods("POST")
	r.HandleFunc("/events/{id}", updateEvent).Methods("PUT")
	r.HandleFunc("/events/{id}", deleteEvent).Methods("DELETE")

	r.HandleFunc("/tasks", getTasks).Methods("GET")
	r.HandleFunc("/tasks", createTask).Methods("POST")
	r.HandleFunc("/tasks/{id}", updateTask).Methods("PUT")
	r.HandleFunc("/tasks/{id}", deleteTask).Methods("DELETE")

	r.HandleFunc("/calendars", getCalendars).Methods("GET")
	r.HandleFunc("/calendars", createCalendar).Methods("POST")
	r.HandleFunc("/calendars/{id}", updateCalendar).Methods("PUT")
	r.HandleFunc("/calendars/{id}", deleteCalendar).Methods("DELETE")

	fmt.Println("Server running on 0.0.0.0:8080")

	log.Println("All Users:")
	log.Printf("%+v", GetUsers())

	if environment == "development" {
		corsMiddleware := handlers.CORS(
			handlers.AllowedOrigins([]string{"http://localhost:5173", "http://localhost:4173"}),
			handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "OPTIONS"}),
			handlers.AllowedHeaders([]string{"Content-Type", "Authorization", "Cookie"}),
			handlers.AllowCredentials(),
		)
		log.Fatal(http.ListenAndServe(":8080", corsMiddleware(r)))
	} else {
		log.Fatal(http.ListenAndServe(":8080", r))
	}
}

func getSession(r *http.Request) *Session {
	if environment == "development" {
		return &Session{
			Active: true,
			Identity: Identity{
				Id:    "b849d4e4-de61-4c27-b6c6-7f2566f7079f",
				State: "active",
				Traits: Traits{
					Email:     "prayuj@prayujt.com",
					FirstName: "Prayuj",
					LastName:  "Tuli",
					Username:  "prayujt",
					Avatar:    "",
				},
			},
		}
	}

	sessionCookie, err := r.Cookie("ory_kratos_session")
	if err != nil {
		return nil
	}
	log.Printf("Session cookie: %s", sessionCookie.Value)

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/sessions/whoami", kratosPublicUrl), nil)
	if err != nil {
		return nil
	}

	req.Header.Set("Cookie", fmt.Sprintf("ory_kratos_session=%s", sessionCookie.Value))
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Error: %v", err)
		log.Printf("Status code: %d", resp.StatusCode)
		return nil
	}
	defer resp.Body.Close()

	var session Session
	if err := json.NewDecoder(resp.Body).Decode(&session); err != nil {
		return nil
	}

	return &session
}
