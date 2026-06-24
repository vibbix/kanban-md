package api

import (
	"context"
	"net/http"

	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/danielgtaylor/huma/v2"
)

type GetTaskInput struct {
	ID int `path:"id" example:"184" doc:"The unique integer ID of the task"`
}

type GetTaskOutput struct {
	Body struct {
		Task *task.Task `json:"task"`
	}
}

// registerTaskRoutes binds all task-related endpoints to the /api/v1/task and /api/v1/tasks groups.
func registerTaskRoutes(v1 huma.API, cfg *config.Config) {
	singularTaskGroup := huma.NewGroup(v1, "/task")
	// pluralTasksGroup := huma.NewGroup(v1, "/tasks") // Will use this for GET /tasks soon

	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "get-task",
		Method:      http.MethodGet,
		Path:        "/{id}", // Resolves to /api/v1/task/{id}
		Summary:     "Get a specific task",
		Description: "Fetches a task from the kanban board by its ID.",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *GetTaskInput) (*GetTaskOutput, error) {
		
		path, err := task.FindByID(cfg.TasksPath(), input.ID)
		if err != nil {
			return nil, huma.Error404NotFound("Task not found", err)
		}

		t, err := task.Read(path)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to read task file", err)
		}

		resp := &GetTaskOutput{}
		resp.Body.Task = t

		return resp, nil
	})
}
