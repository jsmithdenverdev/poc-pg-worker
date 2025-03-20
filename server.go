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
	return mux
}

// addRoutes adds the specified routes to the mux.
func addRoutes(mux *http.ServeMux, pool *pgxpool.Pool) {
	mux.HandleFunc("GET /tasks", listTasks(pool))
	mux.HandleFunc("POST /tasks", createTask(pool))
}
