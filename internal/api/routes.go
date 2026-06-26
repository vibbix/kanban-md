package api

import (
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/danielgtaylor/huma/v2"
)

// RegisterRoutes binds all HTTP paths to their respective handler functions using groups.
func RegisterRoutes(api huma.API, cfg *config.Config) {
	// Base API group
	v1 := huma.NewGroup(api, "/api/v1")

	// Domain-specific groups
	boardGroup := huma.NewGroup(v1, "/board")
	// Note: Meta operations like agent-name and metrics might sit at /meta instead of /api/v1/meta
	metaGroup := huma.NewGroup(api, "/meta")

	// Delegate route registration to the domain handlers
	// We pass the v1 group so the task domain can set up both /task and /tasks
	registerTaskRoutes(v1, cfg)
	registerBoardRoutes(boardGroup, cfg)
	registerMetaRoutes(metaGroup, cfg)
}
