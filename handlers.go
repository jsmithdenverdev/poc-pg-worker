package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// createTask creates a new task.
func createTask(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now()
		task := task{
			ID:      fmt.Sprintf("%d", now.UnixNano()),
			Type:    "default",
			Payload: json.RawMessage(`{"message":"New task created"}`),
			Status:  "pending",
			Created: now,
			Updated: now,
		}

		// Insert task into database (notification will be triggered automatically)
		_, err := pool.Exec(r.Context(),
			"INSERT INTO tasks (id, type, payload, status, created, updated) VALUES ($1, $2, $3, $4, $5, $6)",
			task.ID, task.Type, task.Payload, task.Status, task.Created, task.Updated)
		if err != nil {
			log.Printf("Error inserting task: %v\n", err)
			http.Error(w, "Failed to create task", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(task)
	}
}

// listTasks lists all tasks.
func listTasks(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tasks []task
		// Query tasks from database
		results, err := pool.Query(r.Context(),
			"SELECT id, type, payload, status, created, updated FROM tasks")
		if err != nil {
			if err == pgx.ErrNoRows {
				// No tasks found
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]task{})
				return
			}
			http.Error(w, "failed to read tasks", http.StatusInternalServerError)
			return
		}

		for results.Next() {
			var task task
			err := results.Scan(&task.ID, &task.Type, &task.Payload, &task.Status, &task.Created, &task.Updated)
			if err != nil {
				http.Error(w, "failed to read tasks", http.StatusInternalServerError)
				return
			}
			tasks = append(tasks, task)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(tasks)
	}
}

// createSubscription creates a new subscription.
func createSubscription(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var sub webpush.Subscription
		err := json.NewDecoder(r.Body).Decode(&sub)
		if err != nil {
			http.Error(w, "failed to decode request", http.StatusBadRequest)
			return
		}

		// Store the subscription endpoint in the database
		_, err = pool.Exec(r.Context(),
			"INSERT INTO subscriptions (endpoint, auth, p256dh, created, updated) VALUES ($1, $2, $3, $4, $5)",
			sub.Endpoint, sub.Keys.Auth, sub.Keys.P256dh, time.Now(), time.Now())
		if err != nil {
			http.Error(w, "failed to store subscription", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(sub)
	}
}

// listSubscriptions lists all subscriptions.
func listSubscriptions(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var subs []webpush.Subscription
		// Query subscriptions from database
		results, err := pool.Query(r.Context(),
			"SELECT endpoint, auth, p256dh FROM subscriptions")
		if err != nil {
			if err == pgx.ErrNoRows {
				// No subscriptions found
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]webpush.Subscription{})
				return
			}
			http.Error(w, "failed to read subscriptions", http.StatusInternalServerError)
			return
		}

		for results.Next() {
			var sub webpush.Subscription
			err := results.Scan(&sub.Endpoint, &sub.Keys.Auth, &sub.Keys.P256dh)
			if err != nil {
				http.Error(w, "failed to read subscriptions", http.StatusInternalServerError)
				return
			}
			subs = append(subs, sub)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(subs)
	}
}

// createNotification creates a new notification.
func createNotification(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var not notification
		err := json.NewDecoder(r.Body).Decode(&not)
		if err != nil {
			http.Error(w, "failed to decode request", http.StatusBadRequest)
			return
		}

		now := time.Now()
		not.Created = now
		not.Updated = now

		// Store the notification in the database
		_, err = pool.Exec(r.Context(),
			"INSERT INTO notifications (body, status, created, updated) VALUES ($1, $2, $3, $4)",
			not.Body, "pending", not.Created, not.Updated)
		if err != nil {
			http.Error(w, "failed to store notification", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(not)
	}
}

// listNotifications lists all notifications.
func listNotifications(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var nots []notification
		// Query notifications from database
		results, err := pool.Query(r.Context(),
			"SELECT id, body, created, updated FROM notifications")
		if err != nil {
			if err == pgx.ErrNoRows {
				// No notifications found
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]notification{})
				return
			}
			http.Error(w, "failed to read notifications", http.StatusInternalServerError)
			return
		}

		for results.Next() {
			var not notification
			err := results.Scan(&not.ID, &not.Body, &not.Created, &not.Updated)
			if err != nil {
				http.Error(w, "failed to read notifications", http.StatusInternalServerError)
				return
			}
			nots = append(nots, not)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(nots)
	}
}
