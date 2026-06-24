package api

import (
	"net/http"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func Start(cfg *config.Config, port string) error {
	mux := http.NewServeMux()
	humaAPI := humago.New(mux, huma.DefaultConfig("Kanban-MD API", "1.0.0"))

	// Wire up the routes!
	RegisterRoutes(humaAPI, cfg)

	return http.ListenAndServe(port, mux)
}
