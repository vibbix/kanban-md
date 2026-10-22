package api

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"path/filepath"
	"slices"
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/filelock"
	"github.com/antopolskiy/kanban-md/internal/property"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/danielgtaylor/huma/v2"
)

// -- Structs for GET /task/{id} --
type GetTaskInput struct {
	ID           int      `path:"id" example:"184" doc:"The unique integer ID of the task"`
	ShowProperty []string `query:"show_property,explode" doc:"A custom property key to include in properties; repeatable"`
}

type GetTaskOutput struct {
	Body *TaskResponse
}

// -- Structs for GET /tasks --
type ListTasksOutput struct {
	Body []*TaskResponse
}

// -- Structs for POST /task --
type CreateTaskInput struct {
	Body struct {
		Title      string                     `json:"title" required:"true" doc:"The title of the task"`
		Status     string                     `json:"status,omitempty" doc:"The status column to place the task in"`
		Priority   string                     `json:"priority,omitempty" doc:"The priority level of the task"`
		Class      string                     `json:"class,omitempty" doc:"The class of service for the task"`
		Assignee   string                     `json:"assignee,omitempty" doc:"The person assigned to the task"`
		Tags       []string                   `json:"tags,omitempty" doc:"A list of tags associated with the task"`
		Body       string                     `json:"body,omitempty" doc:"The markdown body content of the task"`
		Due        *date.Date                 `json:"due,omitempty" example:"2025-01-01" doc:"The due date for the task (YYYY-MM-DD)"`
		Estimate   string                     `json:"estimate,omitempty" doc:"The estimated effort for the task"`
		Claimant   string                     `json:"claimant,omitempty" doc:"The agent claiming the task upon creation"`
		Parent     *int                       `json:"parent,omitempty" doc:"The ID of the parent task"`
		DependsOn  []int                      `json:"depends_on,omitempty" doc:"A list of task IDs this task depends on"`
		Properties map[string]json.RawMessage `json:"properties,omitempty" doc:"Custom frontmatter properties; values must be JSON scalars (string, number, boolean, or null)"`
	}
}

type CreateTaskOutput struct {
	Body *TaskResponse
}

// -- Structs for PATCH /task/{id} --
type EditTaskInput struct {
	ID   int `path:"id" doc:"The unique integer ID of the task"`
	Body struct {
		Title           *string                    `json:"title,omitempty" doc:"The updated title of the task"`
		Status          *string                    `json:"status,omitempty" doc:"The updated status of the task"`
		Priority        *string                    `json:"priority,omitempty" doc:"The updated priority of the task"`
		Assignee        *string                    `json:"assignee,omitempty" doc:"The updated assignee"`
		Class           *string                    `json:"class,omitempty" doc:"The updated class of service"`
		BodyText        *string                    `json:"body,omitempty" doc:"The updated markdown body content"`
		Estimate        *string                    `json:"estimate,omitempty" doc:"The updated estimate"`
		Tags            []string                   `json:"tags,omitempty" doc:"The completely replaced list of tags"`
		Due             *date.Date                 `json:"due,omitempty" doc:"The updated due date"`
		Started         *date.Date                 `json:"started,omitempty" doc:"The updated started date"`
		Completed       *date.Date                 `json:"completed,omitempty" doc:"The updated completed date"`
		Parent          *int                       `json:"parent,omitempty" doc:"The updated parent task ID"`
		DependsOn       []int                      `json:"depends_on,omitempty" doc:"The completely replaced list of dependencies"`
		Blocked         *bool                      `json:"blocked,omitempty" doc:"Whether the task is currently blocked"`
		BlockReason     *string                    `json:"block_reason,omitempty" doc:"The reason the task is blocked"`
		Claimant        string                     `json:"claimant,omitempty" doc:"Who is claiming the task to edit it"`
		SetProperties   map[string]json.RawMessage `json:"set_properties,omitempty" doc:"Custom properties to add or replace; values must be JSON scalars"`
		ClearProperties []string                   `json:"clear_properties,omitempty" doc:"Custom property keys to remove"`
	}
}

type EditTaskOutput struct {
	Body *TaskResponse
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
	Task    *TaskResponse `json:"task" doc:"The task in its updated state"`
	Changed bool          `json:"changed" doc:"Whether the status actually changed (false if the task was already at the target status)"`
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
	Task     *TaskResponse `json:"task" doc:"The task in its archived state"`
	Warnings []string      `json:"warnings,omitempty" doc:"Advisory messages, e.g. about tasks that depend on the deleted one"`
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
	Body *TaskResponse
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
	Body *TaskResponse
}

// ListTasksInput holds the query parameters for GET /tasks.
type ListTasksInput struct {
	Search       string   `query:"search" example:"#12" doc:"A number (12 or #12) matches only that exact task ID; anything else is a case-insensitive substring of title, body, or tags"`
	Property     []string `query:"property,explode" example:"sprint=3" doc:"Exact custom property match as KEY=LITERAL (JSON scalar or bare string); repeatable, AND logic"`
	ShowProperty []string `query:"show_property,explode" doc:"A custom property key to include in each task's properties; repeatable"`
}

// -- Structs for GET /context --
type GetContextOutput struct {
	Body ContextResponse
}

// registerTaskRoutes binds all task-related endpoints to the /api/v0/task and /api/v0/tasks groups.
func registerTaskRoutes(v1 huma.API, cfg *config.Config) {
	singularTaskGroup := huma.NewGroup(v1, "/task")
	pluralTasksGroup := huma.NewGroup(v1, "/tasks")
	contextGroup := huma.NewGroup(v1, "/context")

	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "get-task",
		Method:      http.MethodGet,
		Path:        "/{id}",
		Summary:     "Get a specific task",
		Description: "Returns a single task by its integer ID.",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *GetTaskInput) (*GetTaskOutput, error) {
		if err := validatePropertyKeys(input.ShowProperty); err != nil {
			return nil, apiError("Invalid show_property", err)
		}
		path, err := task.FindByID(cfg.TasksPath(), input.ID)
		if err != nil {
			return nil, apiError("Task not found", err)
		}
		t, err := task.Read(path)
		if err != nil {
			return nil, apiError("Failed to read task file", err)
		}
		resp := &GetTaskOutput{}
		resp.Body = toTaskResponse(t, input.ShowProperty...)
		return resp, nil
	})

	huma.Register(pluralTasksGroup, huma.Operation{
		OperationID: "list-tasks",
		Method:      http.MethodGet,
		Path:        "",
		Summary:     "List all tasks",
		Description: "Returns every task on the board, including archived ones, optionally filtered by search.",
		Tags:        []string{"Tasks"},
	}, func(_ context.Context, input *ListTasksInput) (*ListTasksOutput, error) {
		if err := validatePropertyKeys(input.ShowProperty); err != nil {
			return nil, apiError("Invalid show_property", err)
		}
		filters, err := parsePropertyFilters(input.Property)
		if err != nil {
			return nil, apiError("Invalid property filter", err)
		}
		tasks, _, err := board.List(cfg, board.ListOptions{Filter: board.FilterOptions{Search: input.Search, Properties: filters}})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list tasks", err)
		}
		resp := &ListTasksOutput{}
		resp.Body = toTaskResponses(tasks, input.ShowProperty...)
		return resp, nil
	})

	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "create-task",
		Method:      http.MethodPost,
		Path:        "",
		Summary:     "Create a new task",
		Description: "Creates a new task. Status, priority, and class default to the board configuration when omitted.",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *CreateTaskInput) (*CreateTaskOutput, error) {
		edits, err := parsePropertyEdits(input.Body.Properties, nil)
		if err != nil {
			return nil, apiError("Invalid properties", err)
		}
		unlock, err := filelock.Lock(filepath.Join(cfg.Dir(), ".lock"))
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to acquire lock", err)
		}
		defer unlock()

		params := board.CreateParams{
			Properties: edits.sets,
			Title:      input.Body.Title,
			Status:     input.Body.Status,
			Priority:   input.Body.Priority,
			Class:      input.Body.Class,
			Assignee:   input.Body.Assignee,
			Tags:       input.Body.Tags,
			Body:       input.Body.Body,
			Estimate:   input.Body.Estimate,
			Claimant:   input.Body.Claimant,
			Parent:     input.Body.Parent,
			DependsOn:  input.Body.DependsOn,
		}
		if input.Body.Due != nil {
			params.Due = input.Body.Due
		}

		result, err := board.Create(cfg, params, time.Now())
		if err != nil {
			return nil, apiError("Failed to create task", err)
		}

		resp := &CreateTaskOutput{}
		resp.Body = toTaskResponse(result.Task)
		return resp, nil
	})

	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "edit-task",
		Method:      http.MethodPatch,
		Path:        "/{id}",
		Summary:     "Edit a task",
		Description: "Partially updates a task. Only the fields provided in the body are changed. Requires the matching claimant if the task is claimed.",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *EditTaskInput) (*EditTaskOutput, error) {
		edits, err := parsePropertyEdits(input.Body.SetProperties, input.Body.ClearProperties)
		if err != nil {
			return nil, apiError("Invalid properties", err)
		}
		result, err := board.Edit(cfg, input.ID, input.Body.Claimant, false, func(t *task.Task) (bool, error) {
			changed, propErr := edits.apply(t)
			if propErr != nil {
				return false, propErr
			}
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
			return nil, apiError("Failed to edit task", err)
		}

		resp := &EditTaskOutput{}
		resp.Body = toTaskResponse(result.Task)
		return resp, nil
	})

	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "move-task",
		Method:      http.MethodPut,
		Path:        "/{id}/status",
		Summary:     "Move a task",
		Description: "Moves a task to a different status column, enforcing WIP limits and claim requirements. Idempotent if the task is already at the target status.",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *MoveTaskInput) (*MoveTaskOutput, error) {
		if err := task.ValidateStatus(input.Body.Status, cfg.StatusNames()); err != nil {
			return nil, apiError("Invalid status", err)
		}

		params := board.MoveParams{
			ID:        input.ID,
			NewStatus: input.Body.Status,
			Claimant:  input.Body.Claimant,
			SetClaim:  input.Body.SetClaim,
		}
		result, err := board.Move(cfg, params, time.Now())
		if err != nil {
			return nil, apiError("Failed to move task", err)
		}

		resp := &MoveTaskOutput{}
		resp.Body = MoveResultBody{
			Task:    toTaskResponse(result.Task),
			Changed: result.OldStatus != "",
		}
		return resp, nil
	})

	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "delete-task",
		Method:      http.MethodDelete,
		Path:        "/{id}",
		Summary:     "Soft-delete (archive) a task",
		Description: "Soft-deletes a task by moving it to the archived status. Idempotent, and returns advisory warnings about dependent tasks.",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *DeleteTaskInput) (*DeleteTaskOutput, error) {
		result, err := board.Delete(cfg, input.ID, input.Claimant, time.Now())
		if err != nil {
			return nil, apiError("Failed to delete task", err)
		}

		resp := &DeleteTaskOutput{}
		resp.Body = DeleteResultBody{
			Task:     toTaskResponse(result.Task),
			Warnings: result.Warnings,
		}
		return resp, nil
	})

	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "archive-task",
		Method:      http.MethodPost,
		Path:        "/{id}/archive",
		Summary:     "Archive a task",
		Description: "Archives a task by moving it to the archived status. Idempotent if the task is already archived.",
		Tags:        []string{"Tasks"},
	}, func(ctx context.Context, input *ArchiveTaskInput) (*ArchiveTaskOutput, error) {
		// Archive is essentially moving to the archive status
		params := board.MoveParams{
			ID:        input.ID,
			NewStatus: config.ArchivedStatus,
		}
		result, err := board.Move(cfg, params, time.Now())
		if err != nil {
			return nil, apiError("Failed to archive task", err)
		}

		resp := &ArchiveTaskOutput{}
		resp.Body = MoveResultBody{
			Task:    toTaskResponse(result.Task),
			Changed: result.OldStatus != "",
		}
		return resp, nil
	})

	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "pick-task",
		Method:      http.MethodPost,
		Path:        "/{id}/pick",
		Summary:     "Pick a task",
		Description: "Claims a specific task for the given agent.",
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
			return nil, apiError("Failed to pick task", err)
		}

		resp := &PickTaskOutput{}
		resp.Body = toTaskResponse(result.Task)
		return resp, nil
	})

	huma.Register(singularTaskGroup, huma.Operation{
		OperationID: "handoff-task",
		Method:      http.MethodPost,
		Path:        "/{id}/handoff",
		Summary:     "Handoff a task",
		Description: "Hands a task off for review: moves it to the review status, appends an optional note, and releases the claim.",
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
			return nil, apiError("Failed to handoff task", err)
		}

		resp := &HandoffTaskOutput{}
		resp.Body = toTaskResponse(result.Task)
		return resp, nil
	})

	huma.Register(contextGroup, huma.Operation{
		OperationID: "get-context",
		Method:      http.MethodGet,
		Path:        "",
		Summary:     "Get board context",
		Description: "Returns a compact, agent-oriented snapshot of the board: summary counts and grouped task sections.",
		Tags:        []string{"Board"},
	}, func(ctx context.Context, input *struct{}) (*GetContextOutput, error) {
		tasks, _, err := board.List(cfg, board.ListOptions{})
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to list tasks", err)
		}

		contextData := board.GenerateContext(cfg, tasks, board.ContextOptions{}, time.Now())

		resp := &GetContextOutput{}
		resp.Body = toContextResponse(contextData)
		return resp, nil
	})
}

// propertyEdits mirrors the CLI's --set-property/--clear-property plan.
type propertyEdits struct {
	sets   []property.Assignment
	clears []string
}

// parsePropertyEdits validates up front so bad keys or values fail as client
// errors. Sets are sorted because new keys are appended to the task file in
// application order and JSON object order is not preserved.
func parsePropertyEdits(sets map[string]json.RawMessage, clears []string) (propertyEdits, error) {
	edits := propertyEdits{clears: clears}
	for _, key := range slices.Sorted(maps.Keys(sets)) {
		if err := property.ValidateKey(key); err != nil {
			return propertyEdits{}, clierr.New(clierr.InvalidInput, err.Error())
		}
		value, err := property.ParseLiteral(string(sets[key]))
		if err != nil {
			return propertyEdits{}, clierr.Newf(clierr.InvalidInput, "property %q: %v", key, err)
		}
		edits.sets = append(edits.sets, property.Assignment{Key: key, Value: value})
	}
	if err := validatePropertyKeys(clears); err != nil {
		return propertyEdits{}, err
	}
	for _, key := range clears {
		if _, ok := sets[key]; ok {
			return propertyEdits{}, clierr.Newf(clierr.StatusConflict, "cannot set and clear property %q together", key)
		}
	}
	return edits, nil
}

func (e propertyEdits) apply(t *task.Task) (bool, error) {
	changed := false
	for _, set := range e.sets {
		c, err := t.SetPropertyScalar(set.Key, set.Value)
		if err != nil {
			return false, err
		}
		changed = changed || c
	}
	for _, key := range e.clears {
		c, err := t.ClearProperty(key)
		if err != nil {
			return false, err
		}
		changed = changed || c
	}
	return changed, nil
}

func parsePropertyFilters(raw []string) ([]property.Assignment, error) {
	tests := make([]property.Assignment, 0, len(raw))
	for _, r := range raw {
		test, err := property.ParseAssignment(r)
		if err != nil {
			return nil, clierr.New(clierr.InvalidInput, err.Error())
		}
		tests = append(tests, test)
	}
	return tests, nil
}

func validatePropertyKeys(keys []string) error {
	for _, key := range keys {
		if err := property.ValidateKey(key); err != nil {
			return clierr.New(clierr.InvalidInput, err.Error())
		}
	}
	return nil
}
