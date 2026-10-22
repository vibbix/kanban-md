package api_test

import (
	"encoding/json"
	"maps"
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
	resp := hAPI.Post("/api/v0/task", map[string]any{
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
	resp = hAPI.Get("/api/v0/task/1")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var fetched task.Task
	json.Unmarshal(resp.Body.Bytes(), &fetched)
	if fetched.Title != "API Test Task" {
		t.Errorf("expected title 'API Test Task', got %q", fetched.Title)
	}

	// 3. Edit the task
	resp = hAPI.Patch("/api/v0/task/1", map[string]any{
		"title":    "Updated Task",
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
	resp = hAPI.Put("/api/v0/task/1/status", map[string]any{
		"status":   "in-progress",
		"claimant": "test-agent",
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", resp.Code, resp.Body.String())
	}

	// 5. List tasks
	resp = hAPI.Get("/api/v0/tasks")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
	var tasks []*task.Task
	json.Unmarshal(resp.Body.Bytes(), &tasks)
	if len(tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(tasks))
	}

	resp = hAPI.Get("/api/v0/tasks?search=%231")
	if err := json.Unmarshal(resp.Body.Bytes(), &tasks); err != nil || len(tasks) != 1 {
		t.Errorf("search by id: expected 1 task, got %d (%d): %s", len(tasks), resp.Code, resp.Body.String())
	}

	resp = hAPI.Get("/api/v0/tasks?search=nonexistent")
	tasks = nil
	if err := json.Unmarshal(resp.Body.Bytes(), &tasks); err != nil || len(tasks) != 0 {
		t.Errorf("search miss: expected 0 tasks, got %d", len(tasks))
	}

	// 6. Board Config Get/Put
	resp = hAPI.Get("/api/v0/board/config")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	resp = hAPI.Put("/api/v0/board/config", map[string]any{
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
	resp = hAPI.Delete("/api/v0/task/1")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	resp = hAPI.Get("/api/v0/task/1")
	// The task is still accessible, but status should be deleted
	json.Unmarshal(resp.Body.Bytes(), &fetched)
	if fetched.Status != "archived" {
		t.Errorf("expected status 'archived', got %q", fetched.Status)
	}
}

const (
	task1Path  = "/api/v0/task/1"
	propSprint = "sprint"
	propTeam   = "team"
	setProps   = "set_properties"
	clearProps = "clear_properties"
)

// getProperties fetches task 1 and returns its properties as raw JSON literals.
func getProperties(t *testing.T, hAPI humatest.TestAPI, query string) map[string]string {
	t.Helper()
	resp := hAPI.Get(task1Path + query)
	var got struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &got); err != nil {
		t.Fatalf("get %q: %v (%d): %s", query, err, resp.Code, resp.Body.String())
	}
	out := make(map[string]string, len(got.Properties))
	for k, v := range got.Properties {
		out[k] = string(v)
	}
	return out
}

func TestAPIPropertiesRoundTrip(t *testing.T) {
	_, hAPI := setupTestAPI(t)

	resp := hAPI.Post("/api/v0/task", map[string]any{
		"title":      "Props",
		"properties": map[string]any{propSprint: 3, propTeam: "core", "flag": "true"},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("create with properties: got %d: %s", resp.Code, resp.Body.String())
	}
	hAPI.Post("/api/v0/task", map[string]any{"title": "No props"})

	if got := getProperties(t, hAPI, ""); len(got) != 0 {
		t.Errorf("properties must be opt-in, got %v", got)
	}
	got := getProperties(t, hAPI, "?show_property=sprint&show_property=team&show_property=flag&show_property=missing")
	if want := map[string]string{propSprint: `3`, propTeam: `"core"`, "flag": `"true"`}; !maps.Equal(got, want) {
		t.Errorf("properties = %v, want %v", got, want)
	}

	for query, count := range map[string]int{
		"property=team=core":                          1,
		"property=sprint=3&property=team=core":        1,
		"property=sprint=4":                           0,
		"property=flag=true":                          0, // string "true" is not boolean true
		"property=flag=%22true%22&show_property=flag": 1,
	} {
		resp = hAPI.Get("/api/v0/tasks?" + query)
		var tasks []map[string]any
		if err := json.Unmarshal(resp.Body.Bytes(), &tasks); err != nil || len(tasks) != count {
			t.Errorf("%s: expected %d tasks, got %d (%d): %s", query, count, len(tasks), resp.Code, resp.Body.String())
		}
	}

	resp = hAPI.Patch(task1Path, map[string]any{
		setProps:   map[string]any{propSprint: 4, "done": nil},
		clearProps: []string{propTeam},
	})
	if resp.Code != http.StatusOK {
		t.Fatalf("patch properties: got %d: %s", resp.Code, resp.Body.String())
	}
	got = getProperties(t, hAPI, "?show_property=sprint&show_property=team&show_property=done")
	if want := map[string]string{propSprint: "4", "done": "null"}; !maps.Equal(got, want) {
		t.Errorf("after patch = %v, want %v", got, want)
	}
}

func TestAPIPropertyErrors(t *testing.T) {
	_, hAPI := setupTestAPI(t)
	hAPI.Post("/api/v0/task", map[string]any{"title": "Props"})

	for name, tc := range map[string]struct {
		method, path string
		body         any
		code         int
	}{
		"owned key":     {http.MethodPatch, task1Path, map[string]any{setProps: map[string]any{"title": "x"}}, http.StatusBadRequest},
		"owned clear":   {http.MethodPatch, task1Path, map[string]any{clearProps: []string{"status"}}, http.StatusBadRequest},
		"set and clear": {http.MethodPatch, task1Path, map[string]any{setProps: map[string]any{"a": 1}, clearProps: []string{"a"}}, http.StatusConflict},
		"non-scalar":    {http.MethodPost, "/api/v0/task", map[string]any{"title": "t", "properties": map[string]any{"a": []int{1}}}, http.StatusBadRequest},
		"bad filter":    {http.MethodGet, "/api/v0/tasks?property=nokey", nil, http.StatusBadRequest},
		"bad show key":  {http.MethodGet, task1Path + "?show_property=a%20b", nil, http.StatusBadRequest},
		"bad list show": {http.MethodGet, "/api/v0/tasks?show_property=a%20b", nil, http.StatusBadRequest},
	} {
		var args []any
		if tc.body != nil {
			args = append(args, tc.body)
		}
		resp := hAPI.Do(tc.method, tc.path, args...)
		if resp.Code != tc.code {
			t.Errorf("%s: expected %d, got %d: %s", name, tc.code, resp.Code, resp.Body.String())
		}
	}
}

func TestMetaEndpoints(t *testing.T) {
	_, hAPI := setupTestAPI(t)

	// test Agent Name
	resp := hAPI.Get("/api/v0/meta/agent-name")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	// test Version
	resp = hAPI.Get("/api/v0/meta/version")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	// test Metrics
	resp = hAPI.Get("/api/v0/meta/metrics")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}

	// test Logs
	resp = hAPI.Get("/api/v0/meta/logs")
	if resp.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", resp.Code)
	}
}

func TestAPIErrors(t *testing.T) {
	_, hAPI := setupTestAPI(t)

	// GET non-existent task
	resp := hAPI.Get("/api/v0/task/999")
	if resp.Code == http.StatusOK {
		t.Errorf("expected error for non-existent task, got 200")
	}

	// PATCH non-existent task
	resp = hAPI.Patch("/api/v0/task/999", map[string]any{
		"title": "doesn't matter",
	})
	if resp.Code == http.StatusOK {
		t.Errorf("expected error for non-existent task patch, got 200")
	}

	// Create task without title (if title is missing, board.Create might fail or it might create an empty task, let's pass an invalid priority)
	resp = hAPI.Post("/api/v0/task", map[string]any{
		"title":    "bad task",
		"priority": "invalid-priority",
	})
	if resp.Code == http.StatusOK || resp.Code == http.StatusCreated {
		t.Errorf("expected error for invalid priority, got %d", resp.Code)
	}

	// Move task to invalid status
	resp = hAPI.Post("/api/v0/task", map[string]any{"title": "valid task"})
	var created task.Task
	json.Unmarshal(resp.Body.Bytes(), &created)

	resp = hAPI.Put("/api/v0/task/1/status", map[string]any{
		"status": "bogus-status",
	})
	if resp.Code == http.StatusOK {
		t.Errorf("expected error for invalid status, got 200")
	}

	// Delete non-existent task
	resp = hAPI.Delete("/api/v0/task/999")
	if resp.Code == http.StatusOK {
		t.Errorf("expected error for deleting non-existent task, got 200")
	}

	// Board checks
	resp = hAPI.Get("/api/v0/board/check")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for board check, got %d", resp.Code)
	}

	resp = hAPI.Get("/api/v0/board/summary")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for board summary, got %d", resp.Code)
	}

	resp = hAPI.Post("/api/v0/board/compact", map[string]any{})
	if resp.Code == http.StatusOK {
		t.Errorf("expected 501 for compact, got 200")
	}
}

func TestAPIMissingBranches(t *testing.T) {
	_, hAPI := setupTestAPI(t)

	// Create a task to work with
	hAPI.Post("/api/v0/task", map[string]any{"title": "Coverage Task"})

	// 1. PATCH with all optional fields
	resp := hAPI.Patch("/api/v0/task/1", map[string]any{
		"assignee":     "test-user",
		"tags":         []string{"layer-1", "bug"},
		"due":          "2026-12-31",
		"estimate":     "5h",
		"depends_on":   []int{},
		"blocked":      true,
		"block_reason": "waiting for user",
		"body":         "a new body",
	})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for full patch, got %d", resp.Code)
	}

	// 2. Pick task
	resp = hAPI.Post("/api/v0/task/1/pick", map[string]any{
		"agent": "test-agent",
	})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for pick, got %d: %s", resp.Code, resp.Body.String())
	}

	// 3. Handoff task
	resp = hAPI.Post("/api/v0/task/1/handoff", map[string]any{
		"agent": "test-agent",
		"note":  "Here you go",
	})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for handoff, got %d", resp.Code)
	}

	// 4. Archive task
	resp = hAPI.Post("/api/v0/task/1/archive", map[string]any{})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for archive, got %d", resp.Code)
	}

	// 5. Config PUT with all remaining optional fields
	resp = hAPI.Put("/api/v0/board/config", map[string]any{
		"defaults": map[string]string{
			"status":   "todo",
			"priority": "low",
			"class":    "standard",
		},
		"claim_timeout": "2h",
		"tui": map[string]any{
			"title_lines":        3,
			"hide_empty_columns": true,
		},
	})
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for full config put, got %d: %s", resp.Code, resp.Body.String())
	}

	// 6. Meta Logs & Metrics with query params
	resp = hAPI.Get("/api/v0/meta/logs?since=2020-01-01&limit=5&action=create&task=1")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for filtered logs, got %d", resp.Code)
	}

	resp = hAPI.Get("/api/v0/meta/metrics?since=2020-01-01")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for filtered metrics, got %d", resp.Code)
	}

	// 7. Context
	resp = hAPI.Get("/api/v0/context")
	if resp.Code != http.StatusOK {
		t.Errorf("expected 200 for context, got %d", resp.Code)
	}
}
