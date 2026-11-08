package relay

import (
	"context"
	"iter"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// EventsService reads the event log: every event about your tasks and payments, kept 30 days.
type EventsService struct{ c *Client }

// List returns one page of events for your account and the key's mode, newest first by default.
func (s *EventsService) List(ctx context.Context, params *ListEventsParams) (*ListEventsResponse, error) {
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
		if len(params.Types) > 0 {
			q.Set("types", strings.Join(params.Types, ","))
		}
	}
	out := new(ListEventsResponse)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: "/sdk/events", query: q}, out)
}

// ListAll iterates over every matching event, following the cursor page by page.
func (s *EventsService) ListAll(ctx context.Context, params *ListEventsParams) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		page := ListEventsParams{}
		if params != nil {
			page = *params
		}
		for {
			resp, err := s.List(ctx, &page)
			if err != nil {
				yield(Event{}, err)
				return
			}
			for _, event := range resp.Data {
				if !yield(event, nil) {
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
