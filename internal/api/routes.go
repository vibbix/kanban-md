package api

import (
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/danielgtaylor/huma/v2"
)

// RegisterRoutes binds all HTTP paths to their respective handler functions using groups.
func RegisterRoutes(api huma.API, cfg *config.Config) {
	v0 := huma.NewGroup(api, "/api/v0")

	registerTaskRoutes(v0, cfg)
	registerBoardRoutes(huma.NewGroup(v0, "/board"), cfg)
	registerMetaRoutes(huma.NewGroup(v0, "/meta"), cfg)
}
