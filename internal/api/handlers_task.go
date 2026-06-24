package api

import (
	"context"
	"net/http"
	"path/filepath"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/filelock"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/danielgtaylor/huma/v2"
)

// -- Structs for GET /task/{id} --
type GetTaskInput struct {
	ID int `path:"id" example:"184" doc:"The unique integer ID of the task"`
}

type GetTaskOutput struct {
	Body *task.Task
}

// -- Structs for GET /tasks --
type ListTasksOutput struct {
	Body []*task.Task
}

// -- Structs for POST /task --
type CreateTaskInput struct {
	Body struct {
		Title     string     `json:"title" required:"true" doc:"The title of the task"`
		Status    string     `json:"status,omitempty" doc:"The status column to place the task in"`
		Priority  string     `json:"priority,omitempty" doc:"The priority level of the task"`
		Class     string     `json:"class,omitempty" doc:"The class of service for the task"`
		Assignee  string     `json:"assignee,omitempty" doc:"The person assigned to the task"`
		Tags      []string   `json:"tags,omitempty" doc:"A list of tags associated with the task"`
		Body      string     `json:"body,omitempty" doc:"The markdown body content of the task"`
		Due       *date.Date `json:"due,omitempty" example:"2025-01-01" doc:"The due date for the task (YYYY-MM-DD)"`
		Estimate  string     `json:"estimate,omitempty" doc:"The estimated effort for the task"`
		Claimant  string     `json:"claimant,omitempty" doc:"The agent claiming the task upon creation"`
		Parent    *int       `json:"parent,omitempty" doc:"The ID of the parent task"`
		DependsOn []int      `json:"depends_on,omitempty" doc:"A list of task IDs this task depends on"`
	}
}

type CreateTaskOutput struct {
	Body *task.Task
}

// -- Structs for PATCH /task/{id} --
type EditTaskInput struct {
	ID   int `path:"id" doc:"The unique integer ID of the task"`
	Body struct {
		Title       *string    `json:"title,omitempty" doc:"The updated title of the task"`
		Status      *string    `json:"status,omitempty" doc:"The updated status of the task"`
		Priority    *string    `json:"priority,omitempty" doc:"The updated priority of the task"`
		Assignee    *string    `json:"assignee,omitempty" doc:"The updated assignee"`
		Class       *string    `json:"class,omitempty" doc:"The updated class of service"`
		BodyText    *string    `json:"body,omitempty" doc:"The updated markdown body content"`
		Estimate    *string    `json:"estimate,omitempty" doc:"The updated estimate"`
		Tags        []string   `json:"tags,omitempty" doc:"The completely replaced list of tags"`
		Due         *date.Date `json:"due,omitempty" doc:"The updated due date"`
		Started     *date.Date `json:"started,omitempty" doc:"The updated started date"`
		Completed   *date.Date `json:"completed,omitempty" doc:"The updated completed date"`
		Parent      *int       `json:"parent,omitempty" doc:"The updated parent task ID"`
		DependsOn   []int      `json:"depends_on,omitempty" doc:"The completely replaced list of dependencies"`
		Blocked     *bool      `json:"blocked,omitempty" doc:"Whether the task is currently blocked"`
		BlockReason *string    `json:"block_reason,omitempty" doc:"The reason the task is blocked"`
		Claimant    string     `json:"claimant,omitempty" doc:"Who is claiming the task to edit it"`
	}
}

type EditTaskOutput struct {
	Body *task.Task
}

// -- Structs for PUT /task/{id}/status --
type MoveTaskInput struct {
	ID   int `path:"id" doc:"The unique integer ID of the task"`
	Body struct {
		Status   string `json:"status" required:"true" doc:"The target status column to move the task to"`
		Claimant string `json:"claimant,omitempty" doc:"The agent claiming the task during the move"`
		SetClaim bool   `json:"set_claim,omitempty" doc:"Whether to forcibly set the claim to the claimant"`
	}
}

type MoveResultBody struct {
	Task    *task.Task `json:"task"`
	Changed bool       `json:"changed"`
}

type MoveTaskOutput struct {
	Body MoveResultBody
}

// -- Structs for DELETE /task/{id} --
type DeleteTaskInput struct {
	ID       int    `path:"id" doc:"The unique integer ID of the task"`
	Claimant string `query:"claimant" doc:"Who is deleting the task"`
}

type DeleteResultBody struct {
	Task     *task.Task `json:"task"`
	Warnings []string   `json:"warnings,omitempty"`
}

type DeleteTaskOutput struct {
	Body DeleteResultBody
}

// -- Structs for POST /task/{id}/archive --
type ArchiveTaskInput struct {
	ID int `path:"id" doc:"The unique integer ID of the task"`
}

type ArchiveTaskOutput struct {
	Body MoveResultBody
}

// -- Structs for POST /task/{id}/pick --
type PickTaskInput struct {
	ID   int `path:"id" doc:"The unique integer ID of the task"`
	Body struct {
		Agent string `json:"agent" required:"true" doc:"The name of the agent picking the task"`
	}
}

type PickTaskOutput struct {
	Body *task.Task
}

// -- Structs for POST /task/{id}/handoff --
type HandoffTaskInput struct {
	ID   int `path:"id" doc:"The unique integer ID of the task"`
	Body struct {
		Agent string `json:"agent" required:"true" doc:"The name of the agent handing off the task"`
		Note  string `json:"note,omitempty" doc:"An optional handoff note to append to the task body"`
	}
}

type HandoffTaskOutput struct {
	Body *task.Task
}

// -- Structs for GET /context --
type GetContextOutput struct {
	Body board.ContextData
}


// registerTaskRoutes binds all task-related endpoints to the /api/v1/task and /api/v1/tasks groups.
func registerTaskRoutes(v1 huma.API, cfg *config.Config) {
	singularTaskGroup := huma.NewGroup(v1, "/task")
	pluralTasksGroup := huma.NewGroup(v1, "/tasks")
	contextGroup := huma.NewGroup(v1, "/context")

	// 1. GET /api/v1/task/{id}
	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "get-task",
		Method:      http.MethodGet,
		Path:        "/{id}",
		Summary:     "Get a specific task",
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
		resp.Body = t
		return resp, nil
	})

	// 2. GET /api/v1/tasks
	huma.Register(pluralTasksGroup, huma.Operation{
		OperationID: "list-tasks",
		Method:      http.MethodGet,
		Path:        "",
		Summary:     "List all tasks",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *struct{}) (*ListTasksOutput, error) {
		tasks, _, err := board.List(cfg, board.ListOptions{})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list tasks", err)
		}
		resp := &ListTasksOutput{}
		resp.Body = tasks
		return resp, nil
	})

	// 3. POST /api/v1/task
	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "create-task",
		Method:      http.MethodPost,
		Path:        "",
		Summary:     "Create a new task",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *CreateTaskInput) (*CreateTaskOutput, error) {
		unlock, err := filelock.Lock(filepath.Join(cfg.Dir(), ".lock"))
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to acquire lock", err)
		}
		defer unlock()

		params := board.CreateParams{
			Title:    input.Body.Title,
			Status:   input.Body.Status,
			Priority: input.Body.Priority,
			Class:    input.Body.Class,
			Assignee: input.Body.Assignee,
			Tags:     input.Body.Tags,
			Body:     input.Body.Body,
			Estimate: input.Body.Estimate,
			Claimant: input.Body.Claimant,
			Parent:   input.Body.Parent,
			DependsOn: input.Body.DependsOn,
		}
		if input.Body.Due != nil {
			params.Due = input.Body.Due
		}

		result, err := board.Create(cfg, params, time.Now())
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to create task", err)
		}

		resp := &CreateTaskOutput{}
		resp.Body = result.Task
		return resp, nil
	})

	// 4. PATCH /api/v1/task/{id}
	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "edit-task",
		Method:      http.MethodPatch,
		Path:        "/{id}",
		Summary:     "Edit a task",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *EditTaskInput) (*EditTaskOutput, error) {
		result, err := board.Edit(cfg, input.ID, input.Body.Claimant, false, func(t *task.Task) (bool, error) {
			changed := false
			if input.Body.Title != nil {
				t.Title = *input.Body.Title
				changed = true
			}
			if input.Body.Status != nil {
				t.Status = *input.Body.Status
				changed = true
			}
			if input.Body.Priority != nil {
				t.Priority = *input.Body.Priority
				changed = true
			}
			if input.Body.Assignee != nil {
				t.Assignee = *input.Body.Assignee
				changed = true
			}
			if input.Body.Class != nil {
				t.Class = *input.Body.Class
				changed = true
			}
			if input.Body.BodyText != nil {
				t.Body = *input.Body.BodyText
				changed = true
			}
			if input.Body.Estimate != nil {
				t.Estimate = *input.Body.Estimate
				changed = true
			}
			if input.Body.Tags != nil {
				t.Tags = input.Body.Tags
				changed = true
			}
			if input.Body.Due != nil {
				d := *input.Body.Due
				t.Due = &d
				changed = true
			}
			if input.Body.Started != nil {
				d := input.Body.Started.Time
				t.Started = &d
				changed = true
			}
			if input.Body.Completed != nil {
				d := input.Body.Completed.Time
				t.Completed = &d
				changed = true
			}
			if input.Body.Parent != nil {
				t.Parent = input.Body.Parent
				changed = true
			}
			if input.Body.DependsOn != nil {
				t.DependsOn = input.Body.DependsOn
				changed = true
			}
			if input.Body.Blocked != nil {
				t.Blocked = *input.Body.Blocked
				changed = true
			}
			if input.Body.BlockReason != nil {
				t.BlockReason = *input.Body.BlockReason
				changed = true
			}
			return changed, nil
		}, time.Now())

		if err != nil {
			return nil, huma.Error400BadRequest("Failed to edit task", err)
		}

		resp := &EditTaskOutput{}
		resp.Body = result.Task
		return resp, nil
	})

	// 5. PUT /api/v1/task/{id}/status
	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "move-task",
		Method:      http.MethodPut,
		Path:        "/{id}/status",
		Summary:     "Move a task",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *MoveTaskInput) (*MoveTaskOutput, error) {
		params := board.MoveParams{
			ID:        input.ID,
			NewStatus: input.Body.Status,
			Claimant:  input.Body.Claimant,
			SetClaim:  input.Body.SetClaim,
		}
		result, err := board.Move(cfg, params, time.Now())
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to move task", err)
		}

		resp := &MoveTaskOutput{}
		resp.Body = MoveResultBody{
			Task:    result.Task,
			Changed: result.OldStatus != "",
		}
		return resp, nil
	})

	// 6. DELETE /api/v1/task/{id}
	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "delete-task",
		Method:      http.MethodDelete,
		Path:        "/{id}",
		Summary:     "Soft-delete (archive) a task",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *DeleteTaskInput) (*DeleteTaskOutput, error) {
		result, err := board.Delete(cfg, input.ID, input.Claimant, time.Now())
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to delete task", err)
		}

		resp := &DeleteTaskOutput{}
		resp.Body = DeleteResultBody{
			Task:     result.Task,
			Warnings: result.Warnings,
		}
		return resp, nil
	})

	// 7. POST /api/v1/task/{id}/archive
	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "archive-task",
		Method:      http.MethodPost,
		Path:        "/{id}/archive",
		Summary:     "Archive a task",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *ArchiveTaskInput) (*ArchiveTaskOutput, error) {
		// Archive is essentially moving to the archive status
		params := board.MoveParams{
			ID:        input.ID,
			NewStatus: config.ArchivedStatus,
		}
		result, err := board.Move(cfg, params, time.Now())
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to archive task", err)
		}

		resp := &ArchiveTaskOutput{}
		resp.Body = MoveResultBody{
			Task:    result.Task,
			Changed: result.OldStatus != "",
		}
		return resp, nil
	})

	// 8. POST /api/v1/task/{id}/pick
	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "pick-task",
		Method:      http.MethodPost,
		Path:        "/{id}/pick",
		Summary:     "Pick a task",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *PickTaskInput) (*PickTaskOutput, error) {
		// Pick (claim) a specific task
		result, err := board.Edit(cfg, input.ID, "", false, func(t *task.Task) (bool, error) {
			t.ClaimedBy = input.Body.Agent
			now := time.Now()
			t.ClaimedAt = &now
			return true, nil
		}, time.Now())
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to pick task", err)
		}

		resp := &PickTaskOutput{}
		resp.Body = result.Task
		return resp, nil
	})

	// 9. POST /api/v1/task/{id}/handoff
	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "handoff-task",
		Method:      http.MethodPost,
		Path:        "/{id}/handoff",
		Summary:     "Handoff a task",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *HandoffTaskInput) (*HandoffTaskOutput, error) {
		// Handoff is move to review + claim release + note
		result, err := board.Edit(cfg, input.ID, input.Body.Agent, false, func(t *task.Task) (bool, error) {
			t.Status = "review"
			if input.Body.Note != "" {
				t.Body = t.Body + "\n\n" + input.Body.Note
			}
			t.ClaimedBy = ""
			t.ClaimedAt = nil
			return true, nil
		}, time.Now())
		if err != nil {
			return nil, huma.Error400BadRequest("Failed to handoff task", err)
		}

		resp := &HandoffTaskOutput{}
		resp.Body = result.Task
		return resp, nil
	})

	// 10. GET /api/v1/context
	huma.Register(contextGroup, huma.Operation{
		OperationID: "get-context",
		Method:      http.MethodGet,
		Path:        "",
		Summary:     "Get board context",
		Tags:        []string{"Board"},
	}, func(ctx context.Context, input *struct{}) (*GetContextOutput, error) {
		tasks, _, err := board.List(cfg, board.ListOptions{})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list tasks", err)
		}

		contextData := board.GenerateContext(cfg, tasks, board.ContextOptions{}, time.Now())

		resp := &GetContextOutput{}
		resp.Body = contextData
		return resp, nil
	})
}
