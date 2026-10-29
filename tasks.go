package relay

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"strconv"
)

// TasksService quotes, creates and manages deliveries.
type TasksService struct{ c *Client }

// CreateOptions are extra settings for Create.
type CreateOptions struct {
	// IdempotencyKey makes retries safe: the same key never creates a second delivery. Use your
	// order ID. When empty, one is generated per call.
	IdempotencyKey string
}

// Quote prices a delivery without creating it. Pass QuoteID to Create to lock the price.
func (s *TasksService) Quote(ctx context.Context, req TaskQuoteRequest) (*TaskQuoteResponse, error) {
	out := new(TaskQuoteResponse)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: "/sdk/tasks/quote", body: req}, out)
}

// Create creates a delivery.
func (s *TasksService) Create(ctx context.Context, req CreateTaskRequest, opts *CreateOptions) (*CreateTaskResponse, error) {
	r := request{method: http.MethodPost, path: "/sdk/tasks", body: req}
	if opts != nil {
		r.idempotencyKey = opts.IdempotencyKey
	}
	out := new(CreateTaskResponse)
	return out, s.c.do(ctx, r, out)
}

// List returns one page of tasks, newest first by default.
func (s *TasksService) List(ctx context.Context, params *ListTasksParams) (*ListTasksResponse, error) {
	q := url.Values{}
	if params != nil {
		if params.Limit > 0 {
			q.Set("limit", strconv.Itoa(params.Limit))
		}
		if params.Order != "" {
			q.Set("order", params.Order)
		}
		if params.Cursor != "" {
			q.Set("cursor", params.Cursor)
		}
	}
	out := new(ListTasksResponse)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: "/sdk/tasks", query: q}, out)
}

// ListAll iterates over every task, following the cursor page by page. Stop early by breaking
// out of the loop.
//
//	for task, err := range client.Tasks.ListAll(ctx, &relay.ListTasksParams{Limit: 50}) {
//		if err != nil { return err }
//		fmt.Println(task.ID)
//	}
func (s *TasksService) ListAll(ctx context.Context, params *ListTasksParams) iter.Seq2[TaskListItem, error] {
	return func(yield func(TaskListItem, error) bool) {
		page := ListTasksParams{}
		if params != nil {
			page = *params
		}
		for {
			resp, err := s.List(ctx, &page)
			if err != nil {
				yield(TaskListItem{}, err)
				return
			}
			for _, task := range resp.Data {
				if !yield(task, nil) {
					return
				}
			}
			if !resp.HasMore || resp.Cursor == nil || *resp.Cursor == "" {
				return
			}
			page.Cursor = *resp.Cursor
		}
	}
}

// Get returns a task, your rating of it, and the rider's live ETA while they are on the way.
func (s *TasksService) Get(ctx context.Context, taskID string) (*GetTaskResponse, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	out := new(GetTaskResponse)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: "/sdk/tasks/" + id}, out)
}

// Cancel cancels a task that hasn't been picked up. req may be nil.
func (s *TasksService) Cancel(ctx context.Context, taskID string, req *CancelTaskRequest) (*CancelTaskResponse, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	r := request{method: http.MethodDelete, path: "/sdk/tasks/" + id + "/cancel"}
	if req != nil {
		r.body = req
	}
	out := new(CancelTaskResponse)
	return out, s.c.do(ctx, r, out)
}

// Assign offers a manual (AutoAssign false) live task to riders from AvailableRiders.
func (s *TasksService) Assign(ctx context.Context, taskID string, req ManualAssignRequest) (*ManualAssignResponse, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	out := new(ManualAssignResponse)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: "/sdk/tasks/" + id + "/assign", body: req}, out)
}

// AvailableRiders lists riders you can offer a manual task to. params may be nil.
func (s *TasksService) AvailableRiders(ctx context.Context, taskID string, params *AvailableRidersParams) (*AvailableRidersResponse, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	q := url.Values{}
	if params != nil && params.Tier > 0 {
		q.Set("tier", strconv.Itoa(params.Tier))
	}
	out := new(AvailableRidersResponse)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: "/sdk/tasks/" + id + "/available-riders", query: q}, out)
}

// Rate rates the rider of a completed task, once. Check Get's Rating first.
func (s *TasksService) Rate(ctx context.Context, taskID string, req RatingRequest) (*SubmitRatingResponse, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	out := new(SubmitRatingResponse)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: "/sdk/tasks/" + id + "/ratings", body: req}, out)
}

// Dispute opens a dispute on a completed task within the dispute window.
func (s *TasksService) Dispute(ctx context.Context, taskID string, req DisputeRequest) (*DisputeTaskResponse, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	out := new(DisputeTaskResponse)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: "/sdk/tasks/" + id + "/dispute", body: req}, out)
}
