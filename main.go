package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

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
}

var kratosBaseUrl string

func main() {
	kratosBaseUrl = os.Getenv("KRATOS_BASE_URL")
	if kratosBaseUrl == "" {
		kratosBaseUrl = "https://idp.prayujt.com"
	}
	databaseUrl := os.Getenv("DATABASE_URL")
	if databaseUrl == "" {
		log.Fatal("DATABASE_URL must be set")
	}

	InitDatabase(databaseUrl)
	log.Println("Connected to database")
	log.Printf("Using Kratos at: %s", kratosBaseUrl)

	r := mux.NewRouter()
	r.HandleFunc("/events", getEvents).Methods("GET")
	r.HandleFunc("/events", postEvent).Methods("POST")
	r.HandleFunc("/events/generate", generateEventInformation).Methods("POST")

	fmt.Println("Server running on 0.0.0.0:8080")
	log.Fatal(http.ListenAndServe(":8080", r))
}

func getSession(r *http.Request) *Session {
	sessionCookie, err := r.Cookie("ory_kratos_session")
	if err != nil {
		return nil
	}
	log.Printf("Session cookie: %s", sessionCookie.Value)

	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/sessions/whoami", kratosBaseUrl), nil)
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
