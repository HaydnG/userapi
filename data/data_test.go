package data

import (
	"testing"
	"time"

	"github.com/bet365/jingo"
)

// TestUserStructMarshal Ensures no errors when marshalling our standard struct
func TestUserStructMarshal(t *testing.T) {

	encoder := jingo.NewStructEncoder(User{})

	buf := jingo.Buffer{}

	user := User{
		ID:        "0d0f9944-d902-4db1-b83b-6b25a61f89e2",
		FirstName: "Razzil",
		LastName:  "Darkbrew",
		Nickname:  "Alchemist",
		Password:  "moneyMoneyM0n3y",
		Email:     "Razzil.Darkbrew@example.com",
		Country:   "UK",
		CreatedAt: time.Date(2024, 6, 16, 17, 32, 28, 213617100, time.UTC),
		UpdatedAt: time.Date(2024, 6, 16, 17, 32, 28, 213617100, time.UTC),
	}

	encoder.Marshal(&user, &buf)
}

// TestUserStructMarshal Ensures no errors when marshalling our standard struct
func TestUserSliceMarshal(t *testing.T) {

	encoder := jingo.NewSliceEncoder([]User{})

	buf := jingo.Buffer{}

	user := []User{
		{
			ID:        "0d0f9944-d902-4db1-b83b-6b25a61f89e2",
			FirstName: "Razzil",
			LastName:  "Darkbrew",
			Nickname:  "Alchemist",
			Password:  "moneyMoneyM0n3y",
			Email:     "Razzil.Darkbrew@example.com",
			Country:   "UK",
			CreatedAt: time.Date(2024, 6, 16, 17, 32, 28, 213617100, time.UTC),
			UpdatedAt: time.Date(2024, 6, 16, 17, 32, 28, 213617100, time.UTC),
		},
		{
			ID:        "a5557cd5-3083-4ecb-a888-71d98ee1e39e",
			FirstName: "Visage",
			LastName:  "joe",
			Nickname:  "aXE",
			Password:  "VERYSEcure3343",
			Email:     "joe.jim@example.com",
			Country:   "UK",
			CreatedAt: time.Date(2024, 6, 16, 17, 32, 28, 213617100, time.UTC),
			UpdatedAt: time.Date(2024, 6, 16, 17, 32, 28, 213617100, time.UTC),
		},
	}

	encoder.Marshal(&user, &buf)
}
