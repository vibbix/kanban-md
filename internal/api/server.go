package api

import (
	"net/http"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humago"
)

func Start(cfg *config.Config, port string) error {
	mux := http.NewServeMux()

	humaCfg := huma.DefaultConfig("Kanban-MD API", "1.0.0")
	// no need for $schema
	humaCfg.CreateHooks = nil
	humaAPI := humago.New(mux, humaCfg)

	// Wire up the routes!
	RegisterRoutes(humaAPI, cfg)

	return http.ListenAndServe(port, mux)
}
