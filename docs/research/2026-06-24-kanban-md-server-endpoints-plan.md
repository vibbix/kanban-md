# Server Endpoints Mapping Plan

## Context

A plan to add server functionality to `kanban-md` by mapping all CLI commands 1:1 to REST endpoints. The endpoints are divided into three categories: "Board/task Operations", "Board Meta Operations", and "Meta Operations". The implementation should strongly prefer dependencies in the Go Standard Library.

## 1. Board/Task Operations

These endpoints handle the core interactions with tasks and context.

| CLI Command / Alias | Target Source File | HTTP Method | Endpoint | Description |
| :--- | :--- | :--- | :--- | :--- |
| `create` (`add`) | [`cmd/create.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/create.go) | `POST` | `/api/v1/task` | Creates a new task. |
| `list` (`ls`) | [`cmd/list.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/list.go) | `GET` | `/api/v1/tasks` | Lists all tasks on the board. |
| `show` | [`cmd/show.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/show.go) | `GET` | `/api/v1/task/{id}` | Fetches details for a specific task. |
| `edit` | [`cmd/edit.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/edit.go) | `PATCH` | `/api/v1/task/{id}` | Modifies properties (body, title, priority, tags) of a task. |
| `move` | [`cmd/move.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/move.go) | `PUT` | `/api/v1/task/{id}/status` | Moves a task to a different status column. |
| `delete` (`rm`) | [`cmd/delete.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/delete.go) | `DELETE`| `/api/v1/task/{id}` | Permanently deletes a task. |
| `archive` | [`cmd/archive.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/archive.go) | `POST` | `/api/v1/task/{id}/archive` | Archives a specific task. |
| `pick` | [`cmd/pick.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/pick.go) | `POST` | `/api/v1/task/{id}/pick` | Claims a task for a specific agent. |
| `handoff` | [`cmd/handoff.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/handoff.go) | `POST` | `/api/v1/task/{id}/handoff` | Hands off a claimed task to another agent. |
| `context` | [`cmd/context.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/context.go) | `GET` | `/api/v1/context` | Views the current board context. |

## 2. Board Meta Operations

These endpoints handle higher-level board management, including operations ensuring data health and configurations.

| CLI Command / Idea | Target Source File | HTTP Method | Endpoint | Description |
| :--- | :--- | :--- | :--- | :--- |
| `board` (`summary`) | [`cmd/board.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/board.go) | `GET` | `/api/v1/board/summary` | Returns a summary/overview of the board's state. |
| `config get` | [`cmd/config.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/config.go) | `GET` | `/api/v1/config` | Retrieves the board configuration. |
| `config set` | [`cmd/config.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/config.go) | `PUT` | `/api/v1/config` | Updates the board configuration. |
| *(Proposed)* `check`| N/A | `GET` | `/api/v1/board/check` | Checks the board for inconsistencies or malformed files. |
| *(Proposed)* `compact`| N/A | `POST` | `/api/v1/board/compact`| Cleans up and compacts the board (e.g. bulk archiving). |

## 3. Meta Operations

These endpoints provide diagnostic and infrastructure information about the server.

| CLI Command / Idea | Target Source File | HTTP Method | Endpoint | Description |
| :--- | :--- | :--- | :--- | :--- |
| `agent-name` | [`cmd/agent_name.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/agent_name.go) | `GET` | `/meta/agent-name` | Generates and returns a random agent name. |
| `log` | [`cmd/log.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/log.go) | `GET` | `/meta/logs` | Retrieves board/server history logs. |
| `metrics` | [`cmd/metrics.go`](file:///home/vibbix/git/antopolskiy/kanban-md/cmd/metrics.go) | `GET` | `/meta/metrics` | Exposes internal task statistics and metrics. |
| *(Proposed)* `version`| N/A | `GET` | `/meta/version` | Returns the `kanban-md` server build version. |
| *(Proposed)* `uptime` | N/A | `GET` | `/meta/uptime` | Returns the server's running uptime. |
| *(Proposed)* `openapi`| Auto-generated | `GET` | `/openapi.json` | Returns the OpenAPI Schema describing this API (provided by Huma). |

## Dependency & Implementation Strategy

**Recommendation: Use `danielgtaylor/huma/v2` with the Go Standard Library Router.**

Since the project relies on **Go 1.25.7**, we have native support for method-based routing and wildcards directly in the standard library (e.g., `http.ServeMux`). 

1. **Routing:** Standard library `net/http` provides all required routing capabilities natively.
2. **OpenAPI Generation:** We will use `Huma` to automatically generate the OpenAPI 3.1 schema at runtime. Huma requires **zero build-step pre-processing**, produces no generated code files, and perfectly synchronizes the documentation directly from the endpoint logic and Go structs.

### Project Layout

To keep the project clean and maintainable, the HTTP and CLI layers should remain strictly separated. The proposed layout for adding the API is:

```text
kanban-md/
├── cmd/
│   └── server.go             <-- Parses CLI flags, loads config, and calls api.Start()
└── internal/
    └── api/
        ├── server.go         <-- Initializes http.ServeMux, Huma, and graceful shutdown logic
        ├── routes.go         <-- A single file that maps HTTP paths to handler functions
        ├── models.go         <-- Shared Huma structs (e.g., standard error responses, pagination)
        ├── handlers_task.go  <-- Request/Response structs & logic for Task operations
        ├── handlers_board.go <-- Request/Response structs & logic for Board operations
        └── handlers_meta.go  <-- Request/Response structs & logic for Meta operations
```

**Benefits of this layout:**
* `routes.go` acts as a single pane of glass to view the entire API surface.
* `cmd/server.go` avoids housing heavy API business logic.
* Huma input/output struct definitions stay cleanly grouped with the logic that acts upon them in their respective `handlers_*.go` files.
