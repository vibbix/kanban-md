package api_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/antopolskiy/kanban-md/internal/api"
	"github.com/antopolskiy/kanban-md/internal/config"
	"github.com/antopolskiy/kanban-md/internal/task"
	"github.com/danielgtaylor/huma/v2/humatest"
)

func setupTestAPI(t *testing.T) (*config.Config, humatest.TestAPI) {
	t.Helper()
	_, humaAPI := humatest.New(t)
	
	dir := t.TempDir()
	cfg, err := config.Init(dir, "API Test Board")
	if err != nil {
		t.Fatalf("failed to init board: %v", err)
	}

	api.RegisterRoutes(humaAPI, cfg)
	return cfg, humaAPI
}

func TestAPILifecycle(t *testing.T) {
	cfg, hAPI := setupTestAPI(t)

	// 1. Create a task
	resp := hAPI.Post("/api/v1/task", map[string]any{
		"title": "API Test Task",
	})
	if resp.Code != http.StatusCreated && resp.Code != http.StatusOK {
		t.Fatalf("expected 2xx, got %d: %s", resp.Code, resp.Body.String())
	}

	var created task.Task
	if err := json.Unmarshal(resp.Body.Bytes(), &created); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if created.Title != "API Test Task" {
		t.Errorf("expected title 'API Test Task', got %q", created.Title)
	}
	if created.ID != 1 {
		t.Errorf("expected ID 1, got %d", created.ID)
	}

	// 2. Get the task
	resp = hAPI.Get("/api/v1/task/1")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var fetched task.Task
	json.Unmarshal(resp.Body.Bytes(), &fetched)
	if fetched.Title != "API Test Task" {
		t.Errorf("expected title 'API Test Task', got %q", fetched.Title)
	}

	// 3. Edit the task
	resp = hAPI.Patch("/api/v1/task/1", map[string]any{
		"title": "Updated Task",
		"priority": "high",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var edited task.Task
	json.Unmarshal(resp.Body.Bytes(), &edited)
	if edited.Title != "Updated Task" {
		t.Errorf("expected title 'Updated Task', got %q", edited.Title)
	}

	// 4. Move the task
	resp = hAPI.Put("/api/v1/task/1/status", map[string]any{
		"status": "in-progress",
		"claimant": "test-agent",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	
	// 5. List tasks
	resp = hAPI.Get("/api/v1/tasks")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var tasks []*task.Task
	json.Unmarshal(resp.Body.Bytes(), &tasks)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	// 6. Board Config Get/Put
	resp = hAPI.Get("/api/v1/board/config")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	
	resp = hAPI.Put("/api/v1/board/config", map[string]any{
		"board": map[string]string{
			"name": "Updated Board",
		},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}
	if cfg.Board.Name != "Updated Board" {
		t.Errorf("expected board name 'Updated Board', got %q", cfg.Board.Name)
	}

	// 7. Delete task
	resp = hAPI.Delete("/api/v1/task/1")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	resp = hAPI.Get("/api/v1/task/1")
	// The task is still accessible, but status should be deleted
	json.Unmarshal(resp.Body.Bytes(), &fetched)
	if fetched.Status != "archived" {
		t.Errorf("expected status 'archived', got %q", fetched.Status)
	}
}

func TestMetaEndpoints(t *testing.T) {
	_, hAPI := setupTestAPI(t)

	// test Agent Name
	resp := hAPI.Get("/meta/agent-name")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	// test Version
	resp = hAPI.Get("/meta/version")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	// test Metrics
	resp = hAPI.Get("/meta/metrics")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	// test Logs
	resp = hAPI.Get("/meta/logs")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestAPIErrors(t *testing.T) {
	_, hAPI := setupTestAPI(t)

	// GET non-existent task
	resp := hAPI.Get("/api/v1/task/999")
	if resp.Code == http.StatusOK {
		t.Errorf("expected error for non-existent task, got 200")
	}

	// PATCH non-existent task
	resp = hAPI.Patch("/api/v1/task/999", map[string]any{
		"title": "doesn't matter",
	})
	if resp.Code == http.StatusOK {
		t.Errorf("expected error for non-existent task patch, got 200")
	}

	// Create task without title (if title is missing, board.Create might fail or it might create an empty task, let's pass an invalid priority)
	resp = hAPI.Post("/api/v1/task", map[string]any{
		"title":    "bad task",
		"priority": "invalid-priority",
	})
	if resp.Code == http.StatusOK || resp.Code == http.StatusCreated {
		t.Errorf("expected error for invalid priority, got %d", resp.Code)
	}

	// Move task to invalid status
	resp = hAPI.Post("/api/v1/task", map[string]any{"title": "valid task"})
	var created task.Task
	json.Unmarshal(resp.Body.Bytes(), &created)

	resp = hAPI.Put("/api/v1/task/1/status", map[string]any{
		"status": "bogus-status",
	})
	if resp.Code == http.StatusOK {
		t.Errorf("expected error for invalid status, got 200")
	}

	// Delete non-existent task
	resp = hAPI.Delete("/api/v1/task/999")
	if resp.Code == http.StatusOK {
		t.Errorf("expected error for deleting non-existent task, got 200")
	}

	// Board checks
	resp = hAPI.Get("/api/v1/board/check")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for board check, got %d", resp.Code)
	}

	resp = hAPI.Get("/api/v1/board/summary")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for board summary, got %d", resp.Code)
	}

	resp = hAPI.Post("/api/v1/board/compact", map[string]any{})
	if resp.Code == http.StatusOK {
		t.Errorf("expected 501 for compact, got 200")
	}
}

func TestAPIMissingBranches(t *testing.T) {
	_, hAPI := setupTestAPI(t)

	// Create a task to work with
	hAPI.Post("/api/v1/task", map[string]any{"title": "Coverage Task"})

	// 1. PATCH with all optional fields
	resp := hAPI.Patch("/api/v1/task/1", map[string]any{
		"assignee": "test-user",
		"tags": []string{"layer-1", "bug"},
		"due": "2026-12-31",
		"estimate": "5h",
		"depends_on": []int{},
		"blocked": true,
		"block_reason": "waiting for user",
		"body": "a new body",
	})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for full patch, got %d", resp.Code)
	}

	// 2. Pick task
	resp = hAPI.Post("/api/v1/task/1/pick", map[string]any{
		"agent": "test-agent",
	})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for pick, got %d: %s", resp.Code, resp.Body.String())
	}

	// 3. Handoff task
	resp = hAPI.Post("/api/v1/task/1/handoff", map[string]any{
		"agent": "test-agent",
		"note": "Here you go",
	})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for handoff, got %d", resp.Code)
	}

	// 4. Archive task
	resp = hAPI.Post("/api/v1/task/1/archive", map[string]any{})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for archive, got %d", resp.Code)
	}

	// 5. Config PUT with all remaining optional fields
	resp = hAPI.Put("/api/v1/board/config", map[string]any{
		"defaults": map[string]string{
			"status": "todo",
			"priority": "low",
			"class": "standard",
		},
		"claim_timeout": "2h",
		"tui": map[string]any{
			"title_lines": 3,
			"hide_empty_columns": true,
		},
	})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for full config put, got %d: %s", resp.Code, resp.Body.String())
	}

	// 6. Meta Logs & Metrics with query params
	resp = hAPI.Get("/meta/logs?since=2020-01-01&limit=5&action=create&task=1")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for filtered logs, got %d", resp.Code)
	}

	resp = hAPI.Get("/meta/metrics?since=2020-01-01")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for filtered metrics, got %d", resp.Code)
	}

	// 7. Context
	resp = hAPI.Get("/api/v1/context")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for context, got %d", resp.Code)
	}
}
