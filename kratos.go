package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type User struct {
	Id        string `json:"id"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Username  string `json:"username"`
	Avatar    string `json:"avatar"`
}

func GetUsers() []User {
	if environment == "development" {
		return []User{
			{
				Id:        "b849d4e4-de61-4c27-b6c6-7f2566f7079f",
				Email:     "prayuj@prayujt.com",
				FirstName: "Prayuj",
				LastName:  "Tuli",
				Username:  "prayujt",
				Avatar:    "https://static.prayujt.com/images/PRAYUJ.jpg",
			},
		}
	}
	client := &http.Client{}
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/admin/identities", kratosAdminUrl), nil)
	if err != nil {
		return nil
	}

	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		log.Printf("Error: %v", err)
		log.Printf("Status code: %d", resp.StatusCode)
		return nil
	}
	defer resp.Body.Close()

	var identities []Identity
	if err := json.NewDecoder(resp.Body).Decode(&identities); err != nil {
		return nil
	}

	var users []User
	for _, identity := range identities {
		users = append(users, User{
			Id:        identity.Id,
			Email:     identity.Traits.Email,
			FirstName: identity.Traits.FirstName,
			LastName:  identity.Traits.LastName,
			Username:  identity.Traits.Username,
			Avatar:    identity.Traits.Avatar,
		})
	}
	return users
}

func getUsers(w http.ResponseWriter, r *http.Request) {
	session := getSession(r)
	if session == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	if !session.Active {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	users := GetUsers()
	if users == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(users)
}
