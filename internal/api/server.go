package api

import (
	"net/http"
	"time"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

// readHeaderTimeout bounds how long the server waits to read request headers.
const readHeaderTimeout = 10 * time.Second

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "*")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// Start runs the REST API server on the given port, blocking until it exits.
func Start(cfg *config.Config, port string) error {
	mux := http.NewServeMux()

	humaCfg := huma.DefaultConfig("Kanban-MD API", "1.0.0")
	// no need for $schema
	humaCfg.CreateHooks = nil
	humaAPI := humago.New(mux, humaCfg)

	// Wire up the routes!
	RegisterRoutes(humaAPI, cfg)

	srv := &http.Server{
		Addr:              port,
		Handler:           corsMiddleware(mux),
		ReadHeaderTimeout: readHeaderTimeout,
	}
	return srv.ListenAndServe()
}

