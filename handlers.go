package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// createTask returns an HTTP handler that creates a new task in the database.
// The handler:
//   - Generates a unique task ID using timestamp
//   - Inserts the task with 'pending' status
//   - Triggers a notification for the worker via Postgres NOTIFY
//   - Returns the created task as JSON
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

// listTasks returns an HTTP handler that retrieves all tasks from the database.
// The handler:
//   - Queries all tasks from the database
//   - Returns an empty array if no tasks exist
//   - Returns tasks as JSON array
//   - Handles database errors appropriately
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
