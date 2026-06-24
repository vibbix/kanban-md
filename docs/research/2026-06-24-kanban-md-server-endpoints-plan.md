# Server Endpoints Mapping Plan

## Context

A plan to add server functionality to `kanban-md` by mapping all CLI commands 1:1 to REST endpoints. The endpoints are divided into three categories: "Board/task Operations", "Board Meta Operations", and "Meta Operations". The implementation should strongly prefer dependencies in the Go Standard Library.

## 1. Board/Task Operations

These endpoints handle the core interactions with tasks and context.

| CLI Command / Alias | HTTP Method | Endpoint | Description |
| :--- | :--- | :--- | :--- |
| `create` (`add`) | `POST` | `/api/v1/tasks` | Creates a new task. |
| `list` (`ls`) | `GET` | `/api/v1/tasks` | Lists all tasks on the board. |
| `show` | `GET` | `/api/v1/tasks/{id}` | Fetches details for a specific task. |
| `edit` | `PATCH` | `/api/v1/tasks/{id}` | Modifies properties (body, title, priority, tags) of a task. |
| `move` | `PUT` | `/api/v1/tasks/{id}/status` | Moves a task to a different status column. |
| `delete` (`rm`) | `DELETE`| `/api/v1/tasks/{id}` | Permanently deletes a task. |
| `archive` | `POST` | `/api/v1/tasks/{id}/archive` | Archives a specific task. |
| `pick` | `POST` | `/api/v1/tasks/{id}/pick` | Claims a task for a specific agent. |
| `handoff` | `POST` | `/api/v1/tasks/{id}/handoff` | Hands off a claimed task to another agent. |
| `context` | `GET` | `/api/v1/context` | Views the current board context. |

## 2. Board Meta Operations

These endpoints handle higher-level board management, including operations ensuring data health and configurations.

| CLI Command / Idea | HTTP Method | Endpoint | Description |
| :--- | :--- | :--- | :--- |
| `init` | `POST` | `/api/v1/board/init` | Initializes a new kanban board directory. |
| `board` (`summary`) | `GET` | `/api/v1/board/summary` | Returns a summary/overview of the board's state. |
| `config get` | `GET` | `/api/v1/config` | Retrieves the board configuration. |
| `config set` | `PUT` | `/api/v1/config` | Updates the board configuration. |
| `skill` | `POST`| `/api/v1/skills` | Manages AI agent skills (`install`, `update`, `show`). |
| *(Proposed)* `check`| `GET` | `/api/v1/board/check` | Checks the board for inconsistencies or malformed files. |
| *(Proposed)* `compact`| `POST` | `/api/v1/board/compact`| Cleans up and compacts the board (e.g. bulk archiving). |

## 3. Meta Operations

These endpoints provide diagnostic and infrastructure information about the server.

| CLI Command / Idea | HTTP Method | Endpoint | Description |
| :--- | :--- | :--- | :--- |
| `agent-name` | `GET` | `/meta/agent-name` | Generates and returns a random agent name. |
| `log` | `GET` | `/meta/logs` | Retrieves board/server history logs. |
| `metrics` | `GET` | `/meta/metrics` | Exposes internal task statistics and metrics. |
| *(Proposed)* `version`| `GET` | `/meta/version` | Returns the `kanban-md` server build version. |
| *(Proposed)* `uptime` | `GET` | `/meta/uptime` | Returns the server's running uptime. |
| *(Proposed)* `openapi`| `GET` | `/openapi.json` | Returns the OpenAPI Schema describing this API. |

## Dependency Considerations: STDLib vs. External Libs

**Recommendation: Use strictly the Go Standard Library (0 external dependencies).**
Since the project relies on **Go 1.25.7**, we have native support for method-based routing and wildcards directly in the standard library (e.g., `http.HandleFunc("GET /api/v1/tasks/{id}", handler)`).

1. **Routing:** Standard library `net/http` provides all required capabilities natively. No need for `chi` or `gorilla/mux`.
2. **JSON Serialization:** The standard `encoding/json` library is perfectly adequate.
3. **OpenAPI Schema:** Hand-write a static `openapi.json` or `openapi.yaml` file and serve it via a simple `http.ServeFile()` endpoint. This maintains 0 dependencies while providing the exact requested functionality without requiring heavy generation tools like `swaggo`.
