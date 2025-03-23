package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/SherClockHolmes/webpush-go"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type NotificationProcessor func(ctx context.Context, notification *pgconn.Notification) error

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

func worker(pool *pgxpool.Pool, logger *slog.Logger, channelName string) func(ctx context.Context, processor NotificationProcessor) error {
	return func(ctx context.Context, processor NotificationProcessor) error {
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
				if err := processor(ctx, notification); err != nil {
					// Log processing error and continue
					fmt.Fprintf(os.Stderr, "error processing notification: %s\n", err)
				}
			}
		}
	}
}

// processTask processes a task received from the database.
func processTask(logger *slog.Logger, pool *pgxpool.Pool) NotificationProcessor {
	return func(ctx context.Context, notification *pgconn.Notification) error {
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
}

func processNotification(cfg config, logger *slog.Logger, pool *pgxpool.Pool, client *http.Client) NotificationProcessor {
	return func(ctx context.Context, pgnotification *pgconn.Notification) error {
		var n notification
		if err := json.Unmarshal([]byte(pgnotification.Payload), &n); err != nil {
			return fmt.Errorf("failed to unmarshal notification: %w", err)
		}

		// Update notification status
		if _, err := pool.Exec(ctx, "UPDATE notifications SET status = 'processing' WHERE id = $1", n.ID); err != nil {
			return fmt.Errorf("failed to update notification status: %w", err)
		}

		var subscriptions []webpush.Subscription
		// Retrieve all subscriptions
		rows, err := pool.Query(ctx, "SELECT endpoint, auth, p256dh FROM subscriptions")
		if err != nil {
			return fmt.Errorf("failed to retrieve subscriptions: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var s webpush.Subscription
			if err := rows.Scan(&s.Endpoint, &s.Keys.Auth, &s.Keys.P256dh); err != nil {
				return fmt.Errorf("failed to scan subscription: %w", err)
			}
			subscriptions = append(subscriptions, s)
		}

		for _, sub := range subscriptions {
			response, err := webpush.SendNotification([]byte(pgnotification.Payload), &sub, &webpush.Options{
				Subscriber:      "https://pager.com",
				VAPIDPublicKey:  cfg.VapidPublicKey,
				VAPIDPrivateKey: cfg.VapidPrivateKey,
			})
			if err != nil {
				if _, err := pool.Exec(ctx, "UPDATE notifications SET status = 'failed' WHERE id = $1", n.ID); err != nil {
					return fmt.Errorf("failed to update notification status: %w", err)
				}
				return fmt.Errorf("failed to send notification: %w", err)
			}

			defer response.Body.Close()
			body, err := io.ReadAll(response.Body)
			if err != nil {
				return fmt.Errorf("failed to read vapid response body: %w", err)
			}
			logger.InfoContext(ctx, "Notification sent", slog.Any("status", response.Status), slog.Any("body", string(body)))
		}

		// Update notification status
		if _, err := pool.Exec(ctx, "UPDATE notifications SET status = 'completed' WHERE id = $1", n.ID); err != nil {
			return fmt.Errorf("failed to update notification status: %w", err)
		}

		return nil
	}
}
