package api

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

// Duration is a span of time that serializes as an ISO 8601 duration string
// (e.g. "PT2H30M") at the API boundary, instead of a raw number of hours.
type Duration time.Duration

// durationFromHours converts an optional count of hours (as produced by the
// metrics layer) into an optional Duration.
func durationFromHours(h *float64) *Duration {
	if h == nil {
		return nil
	}
	d := Duration(time.Duration(*h * float64(time.Hour)))
	return &d
}

// MarshalJSON renders the duration as an ISO 8601 duration string.
func (d Duration) MarshalJSON() ([]byte, error) {
	return json.Marshal(iso8601Duration(time.Duration(d)))
}

// Schema declares the OpenAPI representation: a string using the standard
// "duration" format. Implementing huma.SchemaProvider keeps the generated
// schema in sync with MarshalJSON.
func (Duration) Schema(huma.Registry) *huma.Schema {
	return &huma.Schema{
		Type:        huma.TypeString,
		Format:      "duration",
		Description: "ISO 8601 duration (e.g. PT2H30M)",
		Examples:    []any{"PT2H30M"},
	}
}

// iso8601Duration formats a duration using hours, minutes, and seconds
// (e.g. "PT1H5M30S"). Sub-second precision is truncated.
func iso8601Duration(d time.Duration) string {
	if d == 0 {
		return "PT0S"
	}
	var b strings.Builder
	if d < 0 {
		b.WriteByte('-')
		d = -d
	}
	b.WriteString("PT")
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	wrote := false
	if h > 0 {
		fmt.Fprintf(&b, "%dH", h)
		wrote = true
	}
	if m > 0 {
		fmt.Fprintf(&b, "%dM", m)
		wrote = true
	}
	if s > 0 || !wrote {
		fmt.Fprintf(&b, "%dS", s)
	}
	return b.String()
}
