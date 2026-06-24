package api

import (
	"context"
	"net/http"
	"time"

	"github.com/danielgtaylor/huma/v2"

	"github.com/antopolskiy/kanban-md/internal/agentname"
	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// Version is the API version string, typically overridden by ldflags.
var Version = "dev"
var startTime = time.Now()

type AgentNameOutputBody map[string]string

type AgentNameOutput struct {
	Body AgentNameOutputBody
}

type LogsInput struct {
	Since  *date.Date `query:"since" doc:"Show entries after this date (YYYY-MM-DD)"`
	Limit  int    `query:"limit" doc:"Maximum number of entries to show"`
	Action string `query:"action" doc:"Filter by action type (create, move, edit, delete, block, unblock)"`
	TaskID int    `query:"task" doc:"Filter by task ID"`
}

type LogsOutput struct {
	Body []board.LogEntry
}

type MetricsInput struct {
	Since *date.Date `query:"since" doc:"Only include tasks completed after this date (YYYY-MM-DD)"`
}

type MetricsOutput struct {
	Body board.Metrics
}

type VersionOutputBody map[string]string

type VersionOutput struct {
	Body VersionOutputBody
}



func registerMetaRoutes(api huma.API, cfg *config.Config) {
	huma.Register(api, huma.Operation{
		OperationID: "get-agent-name",
		Method:      http.MethodGet,
		Path:        "/meta/agent-name",
		Summary:     "Get agent name",
		Description: "Generate a random two-word name suitable for use with --claim.",
		Tags:        []string{"Meta"},
	}, func(ctx context.Context, input *struct{}) (*AgentNameOutput, error) {
		name, err := agentname.Generate()
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to generate agent name", err)
		}
		resp := &AgentNameOutput{}
		resp.Body = map[string]string{"name": name}
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-logs",
		Method:      http.MethodGet,
		Path:        "/meta/logs",
		Summary:     "Get activity logs",
		Description: "Displays the activity log of board mutations.",
		Tags:        []string{"Meta"},
	}, func(ctx context.Context, input *LogsInput) (*LogsOutput, error) {
		opts := board.LogFilterOptions{}
		if input.Since != nil {
			opts.Since = input.Since.Time
		}
		if input.Limit > 0 {
			opts.Limit = input.Limit
		}
		if input.Action != "" {
			opts.Action = input.Action
		}
		if input.TaskID > 0 {
			opts.TaskID = input.TaskID
		}

		entries, err := board.ReadLog(cfg.Dir(), opts)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to read log", err)
		}

		if entries == nil {
			entries = []board.LogEntry{}
		}

		resp := &LogsOutput{
			Body: entries,
		}
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-metrics",
		Method:      http.MethodGet,
		Path:        "/meta/metrics",
		Summary:     "Get flow metrics",
		Description: "Displays flow metrics: throughput, average lead/cycle time, flow efficiency, etc.",
		Tags:        []string{"Meta"},
	}, func(ctx context.Context, input *MetricsInput) (*MetricsOutput, error) {
		allTasks, _, err := task.ReadAllLenient(cfg.TasksPath())
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to read tasks", err)
		}

		if allTasks == nil {
			allTasks = []*task.Task{}
		}

		// Exclude archived tasks from metrics
		tasks := make([]*task.Task, 0, len(allTasks))
		for _, t := range allTasks {
			if !cfg.IsArchivedStatus(t.Status) {
				tasks = append(tasks, t)
			}
		}

		if input.Since != nil {
			sinceTime := input.Since.Time
			filtered := make([]*task.Task, 0, len(tasks))
			for _, t := range tasks {
				if t.Completed == nil || t.Completed.After(sinceTime) {
					filtered = append(filtered, t)
				}
			}
			tasks = filtered
		}

		now := time.Now()
		m := board.ComputeMetrics(cfg, tasks, now)

		resp := &MetricsOutput{
			Body: m,
		}
		return resp, nil
	})

	huma.Register(api, huma.Operation{
		OperationID: "get-version",
		Method:      http.MethodGet,
		Path:        "/meta/version",
		Summary:     "Get API version",
		Tags:        []string{"Meta"},
	}, func(ctx context.Context, input *struct{}) (*VersionOutput, error) {
		resp := &VersionOutput{}
		resp.Body = map[string]string{"version": Version}
		return resp, nil
	})

}
