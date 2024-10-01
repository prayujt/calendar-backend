package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

func GetUsers() []Traits {
	if environment == "development" {
		return []Traits{
			{
				Email:     "prayujtuli@hotmail.com",
				FirstName: "Prayuj",
				LastName:  "Tuli",
				Username:  "prayujt",
				Avatar:    "https://www.gravatar.com/avatar/205e460b479e2e5b48aec07710c08d50",
			},
			{
				Email:     "test@example.com",
				FirstName: "Test",
				LastName:  "User",
				Username:  "testuser",
				Avatar:    "https://www.gravatar.com/avatar/205e460b479e2e5b48aec07710c08d50",
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

	var users []Traits
	for _, identity := range identities {
		users = append(users, Traits{
			Email:     identity.Traits.Email,
			FirstName: identity.Traits.FirstName,
			LastName:  identity.Traits.LastName,
			Username:  identity.Traits.Username,
			Avatar:    identity.Traits.Avatar,
		})
	}
	return users
}
