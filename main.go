package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Config struct {
	DatabaseURL string `env:"DATABASE_URL"`
	ServerPort  string `env:"SERVER_PORT"`
}

type Task struct {
	ID      string    `json:"id"`
	Message string    `json:"message"`
	Status  string    `json:"status"`
	Created time.Time `json:"created"`
}

type application struct {
	config Config
	pool   *pgxpool.Pool
	server *http.Server
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	// Context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Load configuration from environment
	cfg := Config{}
	if err := env.Parse(&cfg); err != nil {
		return fmt.Errorf("error loading configuration: %w", err)
	}

	// Create connection pool
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}
	defer pool.Close()

	// Initialize application
	app := &application{
		config: cfg,
		pool:   pool,
		server: &http.Server{
			Addr: ":" + cfg.ServerPort,
		},
	}

	// Set up routes
	mux := http.NewServeMux()
	mux.HandleFunc("GET /tasks", app.listTasks())
	mux.HandleFunc("POST /tasks", app.createTask())
	app.server.Handler = mux

	// Start worker process
	go app.worker(ctx)

	// Handle shutdown signals
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	// Start server
	serverErrors := make(chan error, 1)
	go func() {
		log.Printf("Server starting on port %s\n", cfg.ServerPort)
		serverErrors <- app.server.ListenAndServe()
	}()

	// Wait for shutdown signal or server error
	select {
	case err := <-serverErrors:
		return fmt.Errorf("server error: %w", err)
	case sig := <-shutdown:
		log.Printf("Starting shutdown (%v)\n", sig)

		// Give outstanding requests 5 seconds to complete
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		if err := app.server.Shutdown(shutdownCtx); err != nil {
			app.server.Close()
			return fmt.Errorf("could not stop server gracefully: %w", err)
		}
	}

	return nil
}

// createTask returns an HTTP handler that creates a new task in the database.
// The handler:
//   - Generates a unique task ID using timestamp
//   - Inserts the task with 'pending' status
//   - Triggers a notification for the worker via Postgres NOTIFY
//   - Returns the created task as JSON
func (app *application) createTask() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		task := Task{
			ID:      fmt.Sprintf("%d", time.Now().UnixNano()),
			Message: "New task created",
			Status:  "pending",
			Created: time.Now(),
		}

		// Insert task into database (notification will be triggered automatically)
		_, err := app.pool.Exec(r.Context(),
			"INSERT INTO tasks (id, message, status, created) VALUES ($1, $2, $3, $4)",
			task.ID, task.Message, task.Status, task.Created)
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
func (app *application) listTasks() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var tasks []Task
		// Query tasks from database
		results, err := app.pool.Query(r.Context(),
			"SELECT id, message, status, created FROM tasks")
		if err != nil {
			if err == pgx.ErrNoRows {
				// No tasks found
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]Task{})
				return
			}
			http.Error(w, "failed to read tasks", http.StatusInternalServerError)
			return
		}

		for results.Next() {
			var task Task
			err := results.Scan(&task.ID, &task.Message, &task.Status, &task.Created)
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

// worker runs as a background goroutine processing task notifications.
// It:
//   - Establishes a dedicated database connection for LISTEN/NOTIFY
//   - Subscribes to the 'tasks_channel'
//   - Processes notifications as they arrive
//   - Simulates work with a delay
//   - Handles graceful shutdown via context cancellation
func (app *application) worker(ctx context.Context) {
	conn, err := app.pool.Acquire(ctx)
	if err != nil {
		log.Printf("Error acquiring connection: %v\n", err)
		return
	}
	defer conn.Release()

	_, err = conn.Exec(ctx, "LISTEN tasks_channel")
	if err != nil {
		log.Printf("Error listening to channel: %v\n", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			log.Println("Worker shutting down...")
			return
		default:
			// Create a new context with timeout for each notification wait
			waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			notification, err := conn.Conn().WaitForNotification(waitCtx)
			cancel()

			if err != nil {
				if !errors.Is(err, context.DeadlineExceeded) {
					log.Printf("Error waiting for notification: %v\n", err)
				}
				continue
			}

			log.Printf("Received task notification: %s\n", notification.Payload)
			// Simulate some work
			time.Sleep(2 * time.Second)

			// Update task status
			_, err = app.pool.Exec(ctx,
				"UPDATE tasks SET status = $1 WHERE id = $2",
				"completed", notification.Payload)
			if err != nil {
				log.Printf("Error updating task: %v\n", err)
				continue
			}

			log.Printf("Processed task: %s\n", notification.Payload)
		}
	}
}
