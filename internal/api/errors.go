package api

import (
	"errors"
	"net/http"

	"github.com/antopolskiy/kanban-md/internal/clierr"
	"github.com/danielgtaylor/huma/v2"
)

// apiError converts a domain error into a huma StatusError whose HTTP status is
// derived from the error's clierr code. Errors that do not carry a
// *clierr.Error are treated as internal server errors, so genuine I/O failures
// surface as 5xx rather than being misreported as client errors. msg provides
// operation context (e.g. "Failed to move task"); the underlying message is
// surfaced in the response's errors list.
func apiError(msg string, err error) huma.StatusError {
	var ce *clierr.Error
	if errors.As(err, &ce) {
		return huma.NewError(statusForClierr(ce.Code), msg, err)
	}
	return huma.Error500InternalServerError(msg, err)
}

// statusForClierr maps a clierr code to its HTTP status. Unknown codes (and
// untyped errors) map to 500, since they represent unexpected server-side
// failures rather than client input problems.
func statusForClierr(code string) int {
	switch code {
	case clierr.TaskNotFound, clierr.BoardNotFound, clierr.DependencyNotFound:
		return http.StatusNotFound
	case clierr.TaskClaimed, clierr.ClaimRequired, clierr.WIPLimitExceeded,
		clierr.ClassWIPExceeded, clierr.StatusConflict, clierr.BoardAlreadyExists:
		return http.StatusConflict
	case clierr.InvalidInput, clierr.InvalidStatus, clierr.InvalidPriority,
		clierr.InvalidDate, clierr.InvalidTaskID, clierr.InvalidClass,
		clierr.InvalidGroupBy, clierr.SelfReference, clierr.NoChanges,
		clierr.NothingToPick, clierr.ConfirmationReq, clierr.BoundaryError:
		return http.StatusBadRequest
	default:
		return http.StatusInternalServerError
	}
}
