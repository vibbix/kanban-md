package api

import (
	"time"

	"github.com/antopolskiy/kanban-md/internal/board"
	"github.com/antopolskiy/kanban-md/internal/date"
	"github.com/antopolskiy/kanban-md/internal/task"
)

// TaskResponse is the public representation of a task. It documents every field
// for the OpenAPI schema and decouples the wire format from the domain model.
type TaskResponse struct {
	ID          int        `json:"id" doc:"Unique integer identifier of the task"`
	Title       string     `json:"title" doc:"Short, imperative title of the task"`
	Status      string     `json:"status" doc:"Current status column (e.g. backlog, todo, in-progress, review, done, archived)"`
	Priority    string     `json:"priority" doc:"Priority level (e.g. low, medium, high, critical)"`
	Created     time.Time  `json:"created" doc:"Timestamp when the task was created"`
	Updated     time.Time  `json:"updated" doc:"Timestamp when the task was last modified"`
	Started     *time.Time `json:"started,omitempty" doc:"Timestamp when work on the task began"`
	Completed   *time.Time `json:"completed,omitempty" doc:"Timestamp when the task reached a terminal status"`
	Assignee    string     `json:"assignee,omitempty" doc:"Person or agent the task is assigned to"`
	Tags        []string   `json:"tags,omitempty" doc:"Free-form labels associated with the task"`
	Due         *date.Date `json:"due,omitempty" doc:"Due date in YYYY-MM-DD format"`
	Estimate    string     `json:"estimate,omitempty" doc:"Estimated effort as a free-form string (e.g. \"4h\", \"2d\")"`
	Parent      *int       `json:"parent,omitempty" doc:"ID of the parent task, if this is a subtask"`
	DependsOn   []int      `json:"depends_on,omitempty" doc:"IDs of tasks that must be completed before this one"`
	Blocked     bool       `json:"blocked,omitempty" doc:"Whether the task is currently blocked"`
	BlockReason string     `json:"block_reason,omitempty" doc:"Explanation for why the task is blocked"`
	ClaimedBy   string     `json:"claimed_by,omitempty" doc:"Agent currently holding the claim on this task"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty" doc:"Timestamp when the current claim was acquired"`
	Class       string     `json:"class,omitempty" doc:"Class of service (e.g. standard, expedite, fixed-date, intangible)"`
	Body        string     `json:"body,omitempty" doc:"Markdown body content below the task frontmatter"`
	File        string     `json:"file,omitempty" doc:"Absolute path to the task's markdown file on disk"`
}

// toTaskResponse converts a domain task into its public representation,
// returning nil for a nil input.
func toTaskResponse(t *task.Task) *TaskResponse {
	if t == nil {
		return nil
	}
	return &TaskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Status:      t.Status,
		Priority:    t.Priority,
		Created:     t.Created,
		Updated:     t.Updated,
		Started:     t.Started,
		Completed:   t.Completed,
		Assignee:    t.Assignee,
		Tags:        t.Tags,
		Due:         t.Due,
		Estimate:    t.Estimate,
		Parent:      t.Parent,
		DependsOn:   t.DependsOn,
		Blocked:     t.Blocked,
		BlockReason: t.BlockReason,
		ClaimedBy:   t.ClaimedBy,
		ClaimedAt:   t.ClaimedAt,
		Class:       t.Class,
		Body:        t.Body,
		File:        t.File,
	}
}

// toTaskResponses converts a slice of domain tasks into public representations.
func toTaskResponses(tasks []*task.Task) []*TaskResponse {
	out := make([]*TaskResponse, len(tasks))
	for i, t := range tasks {
		out[i] = toTaskResponse(t)
	}
	return out
}

// OverviewResponse is the public representation of a board summary.
type OverviewResponse struct {
	BoardName  string             `json:"board_name" doc:"Name of the board"`
	TotalTasks int                `json:"total_tasks" doc:"Total number of active (non-archived) tasks"`
	Statuses   []StatusSummaryDTO `json:"statuses" doc:"Per-status task counts, in configured column order"`
	Priorities []PriorityCountDTO `json:"priorities" doc:"Per-priority task counts, in configured priority order"`
	Classes    []ClassCountDTO    `json:"classes,omitempty" doc:"Per-class task counts, present only when classes of service are configured"`
}

// StatusSummaryDTO holds per-status counts.
type StatusSummaryDTO struct {
	Status   string `json:"status" doc:"Status column name"`
	Count    int    `json:"count" doc:"Number of tasks in this status"`
	WIPLimit int    `json:"wip_limit,omitempty" doc:"Configured WIP limit for this status, if any"`
	Blocked  int    `json:"blocked" doc:"Number of blocked tasks in this status"`
	Overdue  int    `json:"overdue" doc:"Number of overdue tasks in this status"`
}

// PriorityCountDTO holds the count of tasks at a priority level.
type PriorityCountDTO struct {
	Priority string `json:"priority" doc:"Priority level name"`
	Count    int    `json:"count" doc:"Number of tasks at this priority"`
}

// ClassCountDTO holds the count of tasks in a class of service.
type ClassCountDTO struct {
	Class string `json:"class" doc:"Class of service name"`
	Count int    `json:"count" doc:"Number of tasks in this class"`
}

// toOverviewResponse converts a domain board overview into its public form.
func toOverviewResponse(o board.Overview) OverviewResponse {
	statuses := make([]StatusSummaryDTO, len(o.Statuses))
	for i, s := range o.Statuses {
		statuses[i] = StatusSummaryDTO(s)
	}
	priorities := make([]PriorityCountDTO, len(o.Priorities))
	for i, p := range o.Priorities {
		priorities[i] = PriorityCountDTO(p)
	}
	var classes []ClassCountDTO
	if len(o.Classes) > 0 {
		classes = make([]ClassCountDTO, len(o.Classes))
		for i, c := range o.Classes {
			classes[i] = ClassCountDTO(c)
		}
	}
	return OverviewResponse{
		BoardName:  o.BoardName,
		TotalTasks: o.TotalTasks,
		Statuses:   statuses,
		Priorities: priorities,
		Classes:    classes,
	}
}

// MetricsResponse is the public representation of board flow metrics. Time-based
// fields are expressed as ISO 8601 durations rather than raw hours.
type MetricsResponse struct {
	Throughput7d   int            `json:"throughput_7d" doc:"Number of tasks completed in the last 7 days"`
	Throughput30d  int            `json:"throughput_30d" doc:"Number of tasks completed in the last 30 days"`
	AvgLeadTime    *Duration      `json:"avg_lead_time,omitempty" doc:"Average lead time from creation to completion"`
	AvgCycleTime   *Duration      `json:"avg_cycle_time,omitempty" doc:"Average cycle time from start to completion"`
	FlowEfficiency *float64       `json:"flow_efficiency,omitempty" doc:"Ratio of active time to total lead time, between 0 and 1"`
	AgingItems     []AgingItemDTO `json:"aging_items,omitempty" doc:"In-progress tasks ordered by how long they have been open"`
}

// AgingItemDTO is a work item that has started but not yet completed.
type AgingItemDTO struct {
	ID     int      `json:"id" doc:"Task ID"`
	Title  string   `json:"title" doc:"Task title"`
	Status string   `json:"status" doc:"Current status of the task"`
	Age    Duration `json:"age" doc:"How long the task has spent in its current status"`
}

// toMetricsResponse converts domain metrics into their public form, turning
// hour-based averages into durations.
func toMetricsResponse(m board.Metrics) MetricsResponse {
	var aging []AgingItemDTO
	if len(m.AgingItems) > 0 {
		aging = make([]AgingItemDTO, len(m.AgingItems))
		for i, a := range m.AgingItems {
			aging[i] = AgingItemDTO{
				ID:     a.ID,
				Title:  a.Title,
				Status: a.Status,
				Age:    *durationFromHours(&a.AgeHours),
			}
		}
	}
	return MetricsResponse{
		Throughput7d:   m.Throughput7d,
		Throughput30d:  m.Throughput30d,
		AvgLeadTime:    durationFromHours(m.AvgLeadTimeHours),
		AvgCycleTime:   durationFromHours(m.AvgCycleTimeHours),
		FlowEfficiency: m.FlowEfficiency,
		AgingItems:     aging,
	}
}

// ContextResponse is the public representation of the board context snapshot.
type ContextResponse struct {
	BoardName string              `json:"board_name" doc:"Name of the board"`
	Summary   ContextSummaryDTO   `json:"summary" doc:"Aggregate board statistics"`
	Sections  []ContextSectionDTO `json:"sections" doc:"Named groups of tasks (e.g. blocked, in-progress)"`
}

// ContextSummaryDTO holds aggregate board statistics for the context view.
type ContextSummaryDTO struct {
	TotalTasks int    `json:"total_tasks" doc:"Total number of tasks"`
	Active     int    `json:"active" doc:"Number of tasks in active statuses"`
	Blocked    int    `json:"blocked" doc:"Number of blocked tasks"`
	Overdue    int    `json:"overdue" doc:"Number of overdue tasks"`
	WIPWarning string `json:"wip_warning,omitempty" doc:"Warning text if any WIP limit is exceeded"`
}

// ContextSectionDTO is a named group of context items.
type ContextSectionDTO struct {
	Name  string           `json:"name" doc:"Section name"`
	Items []ContextItemDTO `json:"items" doc:"Tasks in this section"`
}

// ContextItemDTO is a single task in the context output.
type ContextItemDTO struct {
	ID       int    `json:"id" doc:"Task ID"`
	Title    string `json:"title" doc:"Task title"`
	Status   string `json:"status" doc:"Current status of the task"`
	Priority string `json:"priority" doc:"Priority level of the task"`
	Assignee string `json:"assignee,omitempty" doc:"Person or agent assigned to the task"`
	Note     string `json:"note,omitempty" doc:"Short contextual note about the task"`
}

// toContextResponse converts domain context data into its public form.
func toContextResponse(c board.ContextData) ContextResponse {
	sections := make([]ContextSectionDTO, len(c.Sections))
	for i, s := range c.Sections {
		items := make([]ContextItemDTO, len(s.Items))
		for j, it := range s.Items {
			items[j] = ContextItemDTO(it)
		}
		sections[i] = ContextSectionDTO{Name: s.Name, Items: items}
	}
	return ContextResponse{
		BoardName: c.BoardName,
		Summary:   ContextSummaryDTO(c.Summary),
		Sections:  sections,
	}
}

// ConsistencyReportResponse is the public representation of a board consistency
// check. Unlike task.ConsistencyReport, its warnings serialize cleanly (the
// domain type carries an unexported error value).
type ConsistencyReportResponse struct {
	Warnings []ConsistencyWarning `json:"warnings" doc:"Task files that could not be read cleanly"`
	Repairs  []string             `json:"repairs" doc:"Automatic repairs that were applied during the check"`
}

// ConsistencyWarning describes a single task file that produced a read warning.
type ConsistencyWarning struct {
	File  string `json:"file" doc:"Base filename of the task that produced the warning"`
	Error string `json:"error" doc:"Description of the problem encountered while reading the file"`
}

// toConsistencyReport converts a domain consistency report into its public
// form, rendering each warning's error as a string.
func toConsistencyReport(r task.ConsistencyReport) ConsistencyReportResponse {
	warnings := make([]ConsistencyWarning, 0, len(r.Warnings))
	for _, w := range r.Warnings {
		msg := ""
		if w.Err != nil {
			msg = w.Err.Error()
		}
		warnings = append(warnings, ConsistencyWarning{File: w.File, Error: msg})
	}
	repairs := r.Repairs
	if repairs == nil {
		repairs = []string{}
	}
	return ConsistencyReportResponse{Warnings: warnings, Repairs: repairs}
}
