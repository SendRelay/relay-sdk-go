package relay

import (
	"context"
	"net/http"
)

// ClaimsService files and follows insurance claims on a task, for insured goods lost, stolen or
// damaged after the rider collected them. Test keys insure nothing, so their tasks can't be claimed on.
type ClaimsService struct{ c *Client }

// Claim is an insurance claim, as you see it. Amounts are in kobo; dates are Lagos days (YYYY-MM-DD).
type Claim struct {
	ID        string `json:"id"`
	Reference string `json:"reference"`
	// Status is SUBMITTED, NEEDS_INFO (answer with Respond), WITH_INSURER, APPROVED, DECLINED, PAID or WITHDRAWN.
	Status string `json:"status"`
	// IncidentType is LOST, STOLEN or DAMAGED.
	IncidentType   string          `json:"incidentType"`
	IncidentDate   string          `json:"incidentDate"`
	ClaimedAmount  int64           `json:"claimedAmount"`
	ApprovedAmount *int64          `json:"approvedAmount"`
	Description    string          `json:"description"`
	Files          []ClaimFileName `json:"files"`
	InfoRequest    *string         `json:"infoRequest"`
	DecisionReason *string         `json:"decisionReason"`
	FiledOn        *string         `json:"filedOn"`
	DecidedOn      *string         `json:"decidedOn"`
	PaidOn         *string         `json:"paidOn"`
}

// ClaimFileName is a file on a claim, by the name it was given.
type ClaimFileName struct {
	Name string `json:"name"`
}

// ClaimState is a task's claim, or whether you can file one and until when.
type ClaimState struct {
	Claim   *Claim `json:"claim"`
	CanFile bool   `json:"canFile"`
	// Reason says why not, when CanFile is false and there is no claim.
	Reason *string `json:"reason"`
	// InsuredValue is the most you can claim; nil when the task isn't insured.
	InsuredValue      *int64  `json:"insuredValue"`
	CertificateNumber *string `json:"certificateNumber"`
	// ClosesOn is the last day to file.
	ClosesOn *string `json:"closesOn"`
}

// ClaimFile is a file from CreateUpload, by its Key, with the name to show for it.
type ClaimFile struct {
	Key  string `json:"key"`
	Name string `json:"name"`
}

// FileClaimRequest files a claim. ClaimedAmount is in kobo, up to the task's insured value.
type FileClaimRequest struct {
	IncidentType  string      `json:"incidentType"`
	IncidentDate  string      `json:"incidentDate"`
	ClaimedAmount int64       `json:"claimedAmount"`
	Description   string      `json:"description"`
	Files         []ClaimFile `json:"files,omitempty"`
}

// RespondToClaimRequest answers a request for more information.
type RespondToClaimRequest struct {
	Message string      `json:"message"`
	Files   []ClaimFile `json:"files,omitempty"`
}

// CreateClaimUploadRequest asks for somewhere to upload one file: image/jpeg, image/png, image/webp or application/pdf.
type CreateClaimUploadRequest struct {
	ContentType string `json:"contentType"`
}

// ClaimUpload is where to PUT one file, with Content-Type: ContentType, before ExpiresAt. Then send Key.
type ClaimUpload struct {
	UploadURL   string `json:"uploadUrl"`
	Key         string `json:"key"`
	ContentType string `json:"contentType"`
	MaxFileSize int64  `json:"maxFileSize"`
	ExpiresAt   string `json:"expiresAt"`
}

func claimPath(taskID string) (string, error) {
	id, err := pathID("taskId", taskID)
	if err != nil {
		return "", err
	}
	return "/sdk/tasks/" + id + "/claim", nil
}

// Get returns the task's claim, or whether you can file one.
func (s *ClaimsService) Get(ctx context.Context, taskID string) (*ClaimState, error) {
	path, err := claimPath(taskID)
	if err != nil {
		return nil, err
	}
	out := new(ClaimState)
	return out, s.c.do(ctx, request{method: http.MethodGet, path: path}, out)
}

// CreateUpload returns somewhere to upload one photo or PDF for a claim. PUT the file to UploadURL
// with the same Content-Type, then pass ClaimFile{Key, Name} in Files.
func (s *ClaimsService) CreateUpload(ctx context.Context, taskID string, req CreateClaimUploadRequest) (*ClaimUpload, error) {
	path, err := claimPath(taskID)
	if err != nil {
		return nil, err
	}
	out := new(ClaimUpload)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: path + "/evidence", body: req}, out)
}

// File files a claim: one per task, once it has ended, for up to its insured value.
func (s *ClaimsService) File(ctx context.Context, taskID string, req FileClaimRequest) (*Claim, error) {
	path, err := claimPath(taskID)
	if err != nil {
		return nil, err
	}
	out := new(Claim)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: path, body: req}, out)
}

// Respond answers Relay's request for more information, while the claim is NEEDS_INFO.
func (s *ClaimsService) Respond(ctx context.Context, taskID string, req RespondToClaimRequest) (*Claim, error) {
	path, err := claimPath(taskID)
	if err != nil {
		return nil, err
	}
	out := new(Claim)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: path + "/respond", body: req}, out)
}

// Withdraw withdraws a claim that is still being decided. You can file again afterwards.
func (s *ClaimsService) Withdraw(ctx context.Context, taskID string) (*Claim, error) {
	path, err := claimPath(taskID)
	if err != nil {
		return nil, err
	}
	out := new(Claim)
	return out, s.c.do(ctx, request{method: http.MethodPost, path: path + "/withdraw"}, out)
}
