package relay

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
)

// Errands: a rider queues, buys, pays a bill, checks something or does a task you describe, at one
// or more stops (TaskType ERRAND). Each job is one Stage. See https://docs.sendrelay.com.ng/errands

// Work stage types. PICKUP and DROPOFF keep working on an errand.
const (
	StageTypePurchase   StageType = "PURCHASE"
	StageTypePayment    StageType = "PAYMENT"
	StageTypeQueue      StageType = "QUEUE"
	StageTypeInspection StageType = "INSPECTION"
	StageTypeCustom     StageType = "CUSTOM"
)

// PurchaseItem is one line of a PURCHASE's shopping list. Prices are in kobo.
type PurchaseItem struct {
	Name     string  `json:"name"`
	Quantity *int    `json:"quantity,omitempty"`
	Unit     *string `json:"unit,omitempty"`
	// MaxUnitPrice is the most you pay for one; above it the rider asks you.
	MaxUnitPrice *int64  `json:"maxUnitPrice,omitempty"`
	Note         *string `json:"note,omitempty"`
	// IfUnavailable overrides the stage's: SKIP, SIMILAR or ASK.
	IfUnavailable string `json:"ifUnavailable,omitempty"`
}

// ErrandQuestion is a question the rider answers at the stop.
type ErrandQuestion struct {
	ID string `json:"id,omitempty"`
	// Type is YES_NO, TEXT, NUMBER, PHOTO or CHOICE.
	Type string `json:"type"`
	Text string `json:"text"`
	// Options (CHOICE only): 2 to 10.
	Options  []string `json:"options,omitempty"`
	Required *bool    `json:"required,omitempty"`
}

// OpeningHours is when a place is open, in Lagos time (HH:MM); nil Days means every day.
type OpeningHours struct {
	Days   []string `json:"days,omitempty"`
	Opens  string   `json:"opens"`
	Closes string   `json:"closes"`
}

// MerchantAccount is the shop's account, if you know it: the account the rider pays must match it.
type MerchantAccount struct {
	BankCode      string `json:"bankCode"`
	AccountNumber string `json:"accountNumber"`
}

// StageSimulate (test keys only) scripts what the simulated rider does at a stage.
type StageSimulate struct {
	// Outcome is COMPLETE, FAIL, ASK or OVERRUN.
	Outcome string `json:"outcome"`
	Reason  string `json:"reason,omitempty"`
	Kind    string `json:"kind,omitempty"`
	Spend   *int64 `json:"spend,omitempty"`
	Minutes *int   `json:"minutes,omitempty"`
}

// ApprovalsPolicy is how long the rider waits for your answer (60 to 1,800 s) and what happens if
// nobody answers (DECLINE, or APPROVE for a request that fits inside the held maximum).
type ApprovalsPolicy struct {
	WaitSeconds int    `json:"waitSeconds,omitempty"`
	IfNoAnswer  string `json:"ifNoAnswer,omitempty"`
}

// StageResult is what came back from a work stage. Open its files with Tasks.ViewFile.
type StageResult struct {
	Minutes     int                  `json:"minutes"`
	PaidMinutes int                  `json:"paidMinutes"`
	Photos      []string             `json:"photos"`
	Note        *string              `json:"note"`
	Receipt     *string              `json:"receipt,omitempty"`
	Items       []PurchaseItemResult `json:"items,omitempty"`
	Spend       *int64               `json:"spend,omitempty"`
	Answers     []QuestionAnswer     `json:"answers,omitempty"`
}

// PurchaseItemResult is one shopping-list line as the rider found it.
type PurchaseItemResult struct {
	Name string `json:"name"`
	// Quantity bought.
	Quantity int `json:"quantity"`
	// Status is BOUGHT, SWAPPED or MISSING.
	Status     string  `json:"status"`
	UnitPrice  *int64  `json:"unitPrice"`
	Total      *int64  `json:"total"`
	Substitute *string `json:"substitute,omitempty"`
}

// QuestionAnswer is the rider's answer to one ErrandQuestion: a bool, string or number.
type QuestionAnswer struct {
	ID    string `json:"id"`
	Value any    `json:"value"`
}

// Approval is a question from the rider. Answer it with Tasks.DecideApproval.
type Approval struct {
	ID             string  `json:"id"`
	TaskID         string  `json:"taskId"`
	StageIndex     int     `json:"stageIndex"`
	StageReference *string `json:"stageReference"`
	// Kind is SUBSTITUTE, PRICE_ABOVE_MAX, OVER_SPEND_LIMIT, MORE_TIME or QUESTION.
	Kind string `json:"kind"`
	// Status is PENDING, APPROVED, DECLINED or EXPIRED.
	Status string `json:"status"`
	// Question is the question in plain words, as the rider app shows it.
	Question string         `json:"question"`
	Proposal map[string]any `json:"proposal"`
	Photo    *string        `json:"photo"`
	// Effect is what approving does to the price.
	Effect     ApprovalEffect `json:"effect"`
	ExpiresAt  string         `json:"expiresAt"`
	IfNoAnswer string         `json:"ifNoAnswer"`
	// DecidedBy is api, timeout, rider, staff, user:<id> or link:<id>.
	DecidedBy *string `json:"decidedBy"`
	DecidedAt *string `json:"decidedAt"`
	Note      *string `json:"note"`
	CreatedAt string  `json:"createdAt"`
}

// ApprovalEffect is what approving does to the price.
type ApprovalEffect struct {
	RaisesMaximum bool  `json:"raisesMaximum"`
	SpendChange   int64 `json:"spendChange"`
	MaxTotalAfter int64 `json:"maxTotalAfter"`
}

// ApprovalDecisionRequest answers an Approval. Note is shown to the rider.
type ApprovalDecisionRequest struct {
	// Decision is APPROVE or DECLINE.
	Decision string `json:"decision"`
	Note     string `json:"note,omitempty"`
}

// ListApprovalsResponse is the rider's questions on a task, newest first.
type ListApprovalsResponse struct {
	Approvals []Approval `json:"approvals"`
}

// ErrandSummary is how far an errand got; Outcome (DONE, PARTLY_DONE, NOT_DONE) is set when it ends.
type ErrandSummary struct {
	Outcome   *string `json:"outcome"`
	Completed int     `json:"completed"`
	Failed    int     `json:"failed"`
	Skipped   int     `json:"skipped"`
}

// ErrandAmounts is the expected and most amount of one part of an errand's price.
type ErrandAmounts struct {
	Expected int64 `json:"expected"`
	Max      int64 `json:"max"`
}

// ErrandQuotePricing is the quote of an errand with work stages: a range, split into the service
// and the goods (spend). Total is expected; MaxTotal is held at create.
type ErrandQuotePricing struct {
	Kind     string           `json:"kind"`
	Total    int64            `json:"total"`
	MaxTotal int64            `json:"maxTotal"`
	MinTotal int64            `json:"minTotal"`
	Service  ErrandAmounts    `json:"service"`
	Spend    ErrandAmounts    `json:"spend"`
	Lines    []map[string]any `json:"lines"`
}

// TaskSchedule is when the rider is expected at each stop, if a rider reaches the first at StartsAt.
type TaskSchedule struct {
	StartsAt   string              `json:"startsAt"`
	FinishesAt string              `json:"finishesAt"`
	Stages     []TaskScheduleStage `json:"stages"`
}

// TaskScheduleStage is one stop of a TaskSchedule.
type TaskScheduleStage struct {
	Index        int     `json:"index"`
	ArrivesAt    string  `json:"arrivesAt"`
	LeavesAt     string  `json:"leavesAt"`
	ClosesAt     *string `json:"closesAt,omitempty"`
	SlackMinutes *int    `json:"slackMinutes,omitempty"`
}

// ErrandWarning is something worth a look before you pay; warnings never block. Codes include
// TIGHT_OPENING_HOURS, TIGHT_DELIVER_BY, SPEND_LIMIT_BELOW_LIST, NO_APPROVAL_LISTENER and LONG_WAIT_HOLD.
type ErrandWarning struct {
	Code       string `json:"code"`
	StageIndex *int   `json:"stageIndex,omitempty"`
	Message    string `json:"message"`
}

// ErrandOptions is the errand price list and limits in force for your account. Available is false
// until errands with work stages are switched on for you.
type ErrandOptions struct {
	Available       bool             `json:"available"`
	Currency        string           `json:"currency"`
	StageTypes      []map[string]any `json:"stageTypes"`
	PaidWait        map[string]any   `json:"paidWait"`
	OnSitePerMinute int64            `json:"onSitePerMinute"`
	Handling        map[string]any   `json:"handling"`
	PayWith         struct {
		Available        []string `json:"available"`
		MerchantTransfer struct {
			Enabled bool `json:"enabled"`
			Paused  bool `json:"paused"`
		} `json:"merchantTransfer"`
	} `json:"payWith"`
	Limits    map[string]int64 `json:"limits"`
	Approvals map[string]any   `json:"approvals"`
}

// SkipStageResponse is the task after a skip.
type SkipStageResponse struct {
	Task Task `json:"task"`
}

// FileViewResponse is a link to a photo or receipt, valid for 15 minutes.
type FileViewResponse struct {
	DownloadURL string `json:"downloadUrl"`
	ExpiresAt   string `json:"expiresAt"`
}

// ErrandsService reads the errand price list.
type ErrandsService struct{ c *Client }

// Options returns the stage fees, paid minutes and limits in force for your account.
func (s *ErrandsService) Options(ctx context.Context) (*ErrandOptions, error) {
	out := new(ErrandOptions)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: "/sdk/errands/options"}, out)
}

// Approvals lists the rider's questions on an errand, newest first. Open ones are PENDING.
func (s *TasksService) Approvals(ctx context.Context, taskID string) (*ListApprovalsResponse, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	out := new(ListApprovalsResponse)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: "/sdk/tasks/" + id + "/approvals"}, out)
}

// DecideApproval answers a question once. The same decision again returns it as it stands; a
// different one after it was decided is a 409.
func (s *TasksService) DecideApproval(ctx context.Context, taskID, approvalID string, req ApprovalDecisionRequest) (*Approval, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	approval, err := pathID("approvalId", approvalID)
	if err != nil {
		return nil, err
	}
	out := new(Approval)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: "/sdk/tasks/" + id + "/approvals/" + approval, body: req}, out)
}

// SkipStage drops a work stage the rider has not reached. You are not charged for it, and a drop-off
// left with nothing to deliver is skipped too.
func (s *TasksService) SkipStage(ctx context.Context, taskID string, stageIndex int) (*SkipStageResponse, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	if stageIndex < 0 {
		return nil, &ValidationError{Field: "stageIndex", Message: "stageIndex must be a whole number from 0"}
	}
	out := new(SkipStageResponse)
	path := "/sdk/tasks/" + id + "/stages/" + strconv.Itoa(stageIndex) + "/skip"
	return out, s.c.do(ctx, request{method: http.MethodPost, path: path}, out)
}

// ViewFile returns a 15-minute link to a photo or receipt that a stage's Result or an Approval names.
// Any other file is a 404 FILE_NOT_FOUND.
func (s *TasksService) ViewFile(ctx context.Context, taskID, fileURL string) (*FileViewResponse, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return nil, err
	}
	if fileURL == "" {
		return nil, &ValidationError{Field: "fileUrl", Message: "fileUrl is required"}
	}
	out := new(FileViewResponse)
	body := map[string]string{"fileUrl": fileURL}
	return out, s.c.do(ctx, request{method: http.MethodPost, path: "/sdk/tasks/" + id + "/files/view", body: body}, out)
}

// MarshalJSON writes a PURCHASE's PurchaseItems as its "items".
func (s Stage) MarshalJSON() ([]byte, error) {
	type plain Stage
	if s.Type == StageTypePurchase && s.PurchaseItems != nil {
		return json.Marshal(struct {
			plain
			Items []PurchaseItem `json:"items,omitempty"`
		}{plain(s), s.PurchaseItems})
	}
	return json.Marshal(plain(s))
}

// UnmarshalJSON reads a PURCHASE's "items" into PurchaseItems.
func (s *Stage) UnmarshalJSON(data []byte) error {
	type plain Stage
	var head struct {
		Type StageType `json:"type"`
	}
	if err := json.Unmarshal(data, &head); err != nil {
		return err
	}
	if head.Type != StageTypePurchase {
		return json.Unmarshal(data, (*plain)(s))
	}
	aux := struct {
		*plain
		Items []PurchaseItem `json:"items"`
	}{plain: (*plain)(s)}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	s.PurchaseItems = aux.Items
	return nil
}
