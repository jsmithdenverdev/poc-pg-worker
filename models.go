package main

import (
	"time"
)

type task struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Payload any       `json:"payload"`
	Status  string    `json:"status"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

type notification struct {
	ID      int       `json:"id"`
	Body    string    `json:"body"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}
