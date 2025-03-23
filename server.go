package main

import (
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"
)

// newServer creates a new HTTP server with the specified database connection
// pool. It sets up the server's routes and returns the server instance.
func newServer(pool *pgxpool.Pool) http.Handler {
	mux := http.NewServeMux()
	addRoutes(mux, pool)
	var handler http.Handler = mux
	handler = corsMiddleware(handler)
	return handler
}

// corsMiddleware adds CORS headers to the response.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// addRoutes adds the specified routes to the mux.
func addRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /tasks", listTasks(pool))
	mux.HandleFunc("POST /tasks", createTask(pool))

	mux.HandleFunc("POST /subscriptions", createSubscription(pool))
	mux.HandleFunc("GET /subscriptions", listSubscriptions(pool))
	mux.HandleFunc("POST /notifications", createNotification(pool))
	mux.HandleFunc("GET /notifications", listNotifications(pool))
}
