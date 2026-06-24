package api

import (
	"context"
	"net/http"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/danielgtaylor/huma/v2"
)

// -- Structs for GET /summary --
type GetSummaryOutput struct {
	Body board.Overview
}

// -- Structs for GET /check --
type GetCheckOutput struct {
	Body task.ConsistencyReport
}



// -- Structs for GET /config --
type GetConfigOutput struct {
	Body config.Config
}

type ConfigUpdateRequest struct {
	Board *struct {
		Name        *string `json:"name,omitempty" doc:"The updated name of the board"`
		Description *string `json:"description,omitempty" doc:"The updated description of the board"`
	} `json:"board,omitempty" doc:"Board metadata"`
	Defaults *struct {
		Status   *string `json:"status,omitempty" doc:"The default status for new tasks"`
		Priority *string `json:"priority,omitempty" doc:"The default priority for new tasks"`
		Class    *string `json:"class,omitempty" doc:"The default class of service for new tasks"`
	} `json:"defaults,omitempty" doc:"Default values for new tasks"`
	ClaimTimeout *string `json:"claim_timeout,omitempty" doc:"The time duration after which a claim expires"`
	TUI *struct {
		TitleLines       *int  `json:"title_lines,omitempty" doc:"Number of lines to allocate for task titles in the TUI"`
		HideEmptyColumns *bool `json:"hide_empty_columns,omitempty" doc:"Whether to hide empty status columns in the TUI"`
	} `json:"tui,omitempty" doc:"TUI rendering settings"`
}

type PutConfigInput struct {
	Body ConfigUpdateRequest
}

type PutConfigOutput struct {
	Body config.Config
}

// registerBoardRoutes binds all board-related endpoints to the /api/v1/board group.
func registerBoardRoutes(api huma.API, cfg *config.Config) {
	// 1. GET /summary
	huma.Register(api, huma.Operation{
		OperationID: "board-summary",
		Method:      http.MethodGet,
		Path:        "/summary",
		Summary:     "Get board summary",
		Tags:        []string{"Board"},
	}, func(ctx context.Context, input *struct{}) (*GetSummaryOutput, error) {
		tasks, _, err := task.ReadAllLenient(cfg.TasksPath())
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to read tasks", err)
		}

		// Exclude archived tasks from board display.
		var activeTasks []*task.Task
		for _, t := range tasks {
			if !cfg.IsArchivedStatus(t.Status) {
				activeTasks = append(activeTasks, t)
			}
		}

		summary := board.Summary(cfg, activeTasks, time.Now())

		resp := &GetSummaryOutput{
			Body: summary,
		}
		return resp, nil
	})

	// 2. GET /check
	huma.Register(api, huma.Operation{
		OperationID: "board-check",
		Method:      http.MethodGet,
		Path:        "/check",
		Summary:     "Check board consistency",
		Tags:        []string{"Board"},
	}, func(ctx context.Context, input *struct{}) (*GetCheckOutput, error) {
		report, err := task.EnsureConsistency(cfg)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to check consistency", err)
		}

		resp := &GetCheckOutput{
			Body: report,
		}
		return resp, nil
	})



	// 4. GET /config
	huma.Register(api, huma.Operation{
		OperationID: "board-config-get",
		Method:      http.MethodGet,
		Path:        "/config",
		Summary:     "Get board configuration",
		Tags:        []string{"Board"},
	}, func(ctx context.Context, input *struct{}) (*GetConfigOutput, error) {
		resp := &GetConfigOutput{
			Body: *cfg,
		}
		return resp, nil
	})

	// 5. PUT /config
	huma.Register(api, huma.Operation{
		OperationID: "board-config-update",
		Method:      http.MethodPut,
		Path:        "/config",
		Summary:     "Update board configuration",
		Tags:        []string{"Board"},
	}, func(ctx context.Context, input *PutConfigInput) (*PutConfigOutput, error) {
		// Apply updates selectively
		if input.Body.Board != nil {
			if input.Body.Board.Name != nil {
				cfg.Board.Name = *input.Body.Board.Name
			}
			if input.Body.Board.Description != nil {
				cfg.Board.Description = *input.Body.Board.Description
			}
		}
		if input.Body.Defaults != nil {
			if input.Body.Defaults.Status != nil {
				cfg.Defaults.Status = *input.Body.Defaults.Status
			}
			if input.Body.Defaults.Priority != nil {
				cfg.Defaults.Priority = *input.Body.Defaults.Priority
			}
			if input.Body.Defaults.Class != nil {
				cfg.Defaults.Class = *input.Body.Defaults.Class
			}
		}
		if input.Body.ClaimTimeout != nil {
			if _, err := time.ParseDuration(*input.Body.ClaimTimeout); err != nil {
				return nil, huma.Error400BadRequest("Invalid claim_timeout format", err)
			}
			cfg.ClaimTimeout = *input.Body.ClaimTimeout
		}
		if input.Body.TUI != nil {
			if input.Body.TUI.TitleLines != nil {
				cfg.TUI.TitleLines = *input.Body.TUI.TitleLines
			}
			if input.Body.TUI.HideEmptyColumns != nil {
				cfg.TUI.HideEmptyColumns = *input.Body.TUI.HideEmptyColumns
			}
		}

		if err := cfg.Validate(); err != nil {
			return nil, huma.Error400BadRequest("Invalid configuration", err)
		}

		if err := cfg.Save(); err != nil {
			return nil, huma.Error500InternalServerError("Failed to save configuration", err)
		}

		resp := &PutConfigOutput{
			Body: *cfg,
		}
		return resp, nil
	})
}
