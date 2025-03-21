package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// task represents a task in the database.
type task struct {
	ID      string    `json:"id"`
	Type    string    `json:"type"`
	Payload any       `json:"payload"`
	Status  string    `json:"status"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
}

// processNotification processes a notification received from the database.
// The notification is expected to contain a JSON payload that can be
// unmarshalled into a task struct.
func processNotification(ctx context.Context, logger *slog.Logger, notification *pgconn.Notification, pool *pgxpool.Pool) error {
	var t task
	if err := json.Unmarshal([]byte(notification.Payload), &t); err != nil {
		return fmt.Errorf("failed to unmarshal task: %w", err)
	}

	// Update task status
	if _, err := pool.Exec(ctx, "UPDATE tasks SET status = 'processing' WHERE id = $1", t.ID); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// Process the task here
	// For now, just log it
	logger.InfoContext(ctx, "Processing task", slog.Any("task", t))

	// Update task status
	if _, err := pool.Exec(ctx, "UPDATE tasks SET status = 'completed' WHERE id = $1", t.ID); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	return nil
}

// worker returns a function that starts a worker process to handle notifications
// from the specified channel.
const (
	maxRetries    = 5
	retryInterval = 5 * time.Second
)

func waitForConnection(ctx context.Context, pool *pgxpool.Pool) error {
	for i := 0; i < maxRetries; i++ {
		if err := pool.Ping(ctx); err == nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(retryInterval):
			fmt.Fprintf(os.Stderr, "waiting for database connection (attempt %d/%d)\n", i+1, maxRetries)
		}
	}
	return fmt.Errorf("failed to connect to database after %d attempts", maxRetries)
}

func worker(pool *pgxpool.Pool, logger *slog.Logger, channelName string) func(ctx context.Context) error {
	return func(ctx context.Context) error {
		// Wait for database connection
		if err := waitForConnection(ctx, pool); err != nil {
			return fmt.Errorf("worker failed to connect to database: %w", err)
		}

		// Listen for notifications
		conn, err := pool.Acquire(ctx)
		if err != nil {
			return fmt.Errorf("failed to acquire connection: %w", err)
		}
		defer conn.Release()

		// Start listening
		if _, err := conn.Exec(ctx, fmt.Sprintf("LISTEN %s", channelName)); err != nil {
			return fmt.Errorf("failed to start listening: %w", err)
		}

		for {
			select {
			case <-ctx.Done():
				return nil
			default:
				notification, err := conn.Conn().WaitForNotification(ctx)
				if err != nil {
					if ctx.Err() != nil {
						// Context cancelled, exit cleanly
						return nil
					}
					// Log error and continue
					fmt.Fprintf(os.Stderr, "error waiting for notification: %s\n", err)
					continue
				}

				// Process notification
				if err := processNotification(ctx, logger, notification, pool); err != nil {
					// Log processing error and continue
					fmt.Fprintf(os.Stderr, "error processing notification: %s\n", err)
				}
			}
		}
	}
}
