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
