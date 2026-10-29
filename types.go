package relay

import "encoding/json"

// Amounts are in kobo (100 kobo = ₦1). Timestamps are ISO 8601 strings, as the API sends them.

type TaskType string

const (
	TaskTypePassengerRide   TaskType = "PASSENGER_RIDE"
	TaskTypeFoodDelivery    TaskType = "FOOD_DELIVERY"
	TaskTypePackageDelivery TaskType = "PACKAGE_DELIVERY"
	TaskTypeBulkDelivery    TaskType = "BULK_DELIVERY"
	TaskTypeErrand          TaskType = "ERRAND"
)

type TaskStatus string

const (
	TaskStatusPending       TaskStatus = "PENDING"
	TaskStatusOffered       TaskStatus = "OFFERED"
	TaskStatusAssigned      TaskStatus = "ASSIGNED"
	TaskStatusInProgress    TaskStatus = "IN_PROGRESS"
	TaskStatusCompleted     TaskStatus = "COMPLETED"
	TaskStatusCancelled     TaskStatus = "CANCELLED"
	TaskStatusFailed        TaskStatus = "FAILED"
	TaskStatusPaymentFailed TaskStatus = "PAYMENT_FAILED"
)

type StageType string

const (
	StageTypePickup  StageType = "PICKUP"
	StageTypeDropoff StageType = "DROPOFF"
)

type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Address   string  `json:"address"`
}

type Item struct {
	Name                string  `json:"name"`
	Description         *string `json:"description,omitempty"`
	EstimatedValue      int64   `json:"estimatedValue"`
	EstimatedWeight     string  `json:"estimatedWeight"`
	EstimatedSize       string  `json:"estimatedSize"`
	SpecialInstructions *string `json:"specialInstructions,omitempty"`
}

type Passenger struct {
	Name         string  `json:"name"`
	ContactInfo  *string `json:"contactInfo,omitempty"`
	SpecialNeeds *string `json:"specialNeeds,omitempty"`
	LuggageCount *int    `json:"luggageCount,omitempty"`
}

// Stage is one stop. On requests set Type, Location, Instructions and Items (or Passengers);
// responses also fill in its progress.
type Stage struct {
	Type            StageType   `json:"type"`
	Location        Location    `json:"location"`
	Instructions    string      `json:"instructions,omitempty"`
	Items           []Item      `json:"items,omitempty"`
	Passengers      []Passenger `json:"passengers,omitempty"`
	Status          *string     `json:"status,omitempty"`
	CompletedAt     *string     `json:"completedAt,omitempty"`
	CompletionPhoto *string     `json:"completionPhoto,omitempty"`
	CompletionNotes *string     `json:"completionNotes,omitempty"`
}

type Preferences struct {
	MinRating              *float64 `json:"minRating,omitempty"`
	HasBox                 *bool    `json:"hasBox,omitempty"`
	HasRefrigeration       *bool    `json:"hasRefrigeration,omitempty"`
	HasAirConditioning     *bool    `json:"hasAirConditioning,omitempty"`
	VehicleType            []string `json:"vehicleType,omitempty"`
	AllowedOrganisations   []string `json:"allowedOrganisations,omitempty"`
	BlockedOrganisations   []string `json:"blockedOrganisations,omitempty"`
	AllowIndependentRiders *bool    `json:"allowIndependentRiders,omitempty"`
}

// CompletionCodeRequest asks for proof of delivery: set Code to choose the digits, or Length
// (4 or 6) to have Relay generate them.
type CompletionCodeRequest struct {
	Code   string `json:"code,omitempty"`
	Length int    `json:"length,omitempty"`
}

type TaskQuoteRequest struct {
	TaskType    TaskType     `json:"taskType"`
	Priority    string       `json:"priority,omitempty"`
	Stages      []Stage      `json:"stages"`
	Preferences *Preferences `json:"preferences,omitempty"`
}

type CreateTaskRequest struct {
	TaskType          TaskType               `json:"taskType"`
	Priority          string                 `json:"priority,omitempty"`
	Stages            []Stage                `json:"stages"`
	AutoAssign        *bool                  `json:"autoAssign,omitempty"`
	PaymentMethod     string                 `json:"paymentMethod,omitempty"`
	Notes             string                 `json:"notes,omitempty"`
	ExternalID        string                 `json:"externalId,omitempty"`
	QuoteID           string                 `json:"quoteId,omitempty"`
	Preferences       *Preferences           `json:"preferences,omitempty"`
	CompletionCode    *CompletionCodeRequest `json:"completionCode,omitempty"`
	SimulationOutcome string                 `json:"simulationOutcome,omitempty"`
}

type RiderSnippet struct {
	ID        string          `json:"id"`
	FirstName string          `json:"firstName"`
	LastName  string          `json:"lastName,omitempty"`
	Phone     string          `json:"phone,omitempty"`
	Photo     *string         `json:"photo"`
	Vehicle   json.RawMessage `json:"vehicle,omitempty"`
	Rating    *RatingSummary  `json:"rating,omitempty"`
}

type RatingSummary struct {
	Average      float64 `json:"average"`
	TotalReviews int     `json:"totalReviews"`
}

type TaskRouting struct {
	Provider                 string  `json:"provider"`
	PricingBasis             string  `json:"pricingBasis"`
	GeneratedAt              string  `json:"generatedAt"`
	DistanceMeters           int64   `json:"distanceMeters"`
	DurationSeconds          int64   `json:"durationSeconds"`
	DurationInTrafficSeconds int64   `json:"durationInTrafficSeconds"`
	TollCostKobo             int64   `json:"tollCostKobo"`
	StagesCount              int     `json:"stagesCount"`
	RoutePolyline            *string `json:"routePolyline"`
}

type TaskCompletionCode struct {
	Code           string  `json:"code"`
	Length         int     `json:"length"`
	Source         string  `json:"source"`
	Status         string  `json:"status"`
	FailedAttempts int     `json:"failedAttempts"`
	LockedUntil    *string `json:"lockedUntil"`
	VerifiedAt     *string `json:"verifiedAt"`
}

type Task struct {
	ID                     string              `json:"id"`
	DeveloperID            string              `json:"developerId"`
	RiderID                *string             `json:"riderId"`
	ExternalID             *string             `json:"externalId"`
	Status                 TaskStatus          `json:"status"`
	TaskType               TaskType            `json:"taskType"`
	Priority               string              `json:"priority"`
	Source                 string              `json:"source,omitempty"`
	AutoAssign             bool                `json:"autoAssign"`
	CurrentStageIndex      *int                `json:"currentStageIndex"`
	Notes                  *string             `json:"notes"`
	Preferences            *Preferences        `json:"preferences"`
	Stages                 []Stage             `json:"stages"`
	DeliveryFee            int64               `json:"deliveryFee"`
	TotalFee               int64               `json:"totalFee"`
	PaymentStatus          *string             `json:"paymentStatus"`
	PaymentMethod          string              `json:"paymentMethod"`
	Pricing                map[string]any      `json:"pricing"`
	CreatedAt              string              `json:"createdAt"`
	UpdatedAt              string              `json:"updatedAt"`
	OfferedAt              *string             `json:"offeredAt"`
	AcceptedAt             *string             `json:"acceptedAt"`
	AssignedAt             *string             `json:"assignedAt"`
	CompletedAt            *string             `json:"completedAt"`
	FailedAt               *string             `json:"failedAt"`
	CancelledAt            *string             `json:"cancelledAt"`
	FailureReason          *string             `json:"failureReason"`
	DisputeID              *string             `json:"disputeId"`
	Routing                *TaskRouting        `json:"routing"`
	RoutePolyline          *string             `json:"routePolyline"`
	CompletionCodeRequired bool                `json:"completionCodeRequired"`
	CompletionCodeLength   *int                `json:"completionCodeLength"`
	CompletionCode         *TaskCompletionCode `json:"completionCode,omitempty"`
	Rider                  *RiderSnippet       `json:"rider,omitempty"`
}

// TaskLiveEta is how far the rider is from the stop they are heading to, from the road route
// their app is navigating and live traffic.
type TaskLiveEta struct {
	StageIndex int    `json:"stageIndex"`
	Seconds    int    `json:"seconds"`
	Meters     int    `json:"meters"`
	ArrivesAt  string `json:"arrivesAt"`
	UpdatedAt  string `json:"updatedAt"`
}

// TaskRating is your rating of a completed task's rider.
type TaskRating struct {
	Rating    int     `json:"rating"`
	Comment   *string `json:"comment"`
	CreatedAt string  `json:"createdAt"`
}

type GetTaskResponse struct {
	Task Task `json:"task"`
	// Rating is your rating of this task's rider, or nil if you haven't rated it.
	Rating *TaskRating `json:"rating"`
	// LiveEta is the rider's ETA to their next stop, or nil when no rider is on the way.
	LiveEta *TaskLiveEta `json:"liveEta"`
}

// PricingBreakdown is the price of a delivery. Fees holds every line as the API sent it,
// including fee lines not named here (insuranceFee, temporalFee, ...).
type PricingBreakdown struct {
	Total              int64          `json:"total"`
	Subtotal           int64          `json:"subtotal"`
	Tax                int64          `json:"tax"`
	PlatformFee        int64          `json:"platformFee"`
	PriorityFee        int64          `json:"priorityFee"`
	PricingVehicleType string         `json:"pricingVehicleType,omitempty"`
	SurgeMultiplier    float64        `json:"surgeMultiplier,omitempty"`
	Route              map[string]any `json:"route,omitempty"`
	Fees               map[string]any `json:"-"`
}

func (p *PricingBreakdown) UnmarshalJSON(data []byte) error {
	type plain PricingBreakdown
	if err := json.Unmarshal(data, (*plain)(p)); err != nil {
		return err
	}
	return json.Unmarshal(data, &p.Fees)
}

func (p PricingBreakdown) MarshalJSON() ([]byte, error) {
	type plain PricingBreakdown
	out := map[string]any{}
	for k, v := range p.Fees {
		out[k] = v
	}
	named, err := json.Marshal(plain(p))
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(named, &out); err != nil {
		return nil, err
	}
	return json.Marshal(out)
}

type TaskQuoteResponse struct {
	PricingBreakdown
	QuoteID         string           `json:"quoteId"`
	ExpiresAt       string           `json:"expiresAt"`
	QuoteTTLSeconds int              `json:"quoteTtlSeconds"`
	Task            TaskQuoteRequest `json:"task"`
}

func (q *TaskQuoteResponse) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &q.PricingBreakdown); err != nil {
		return err
	}
	var rest struct {
		QuoteID         string           `json:"quoteId"`
		ExpiresAt       string           `json:"expiresAt"`
		QuoteTTLSeconds int              `json:"quoteTtlSeconds"`
		Task            TaskQuoteRequest `json:"task"`
	}
	if err := json.Unmarshal(data, &rest); err != nil {
		return err
	}
	q.QuoteID, q.ExpiresAt, q.QuoteTTLSeconds, q.Task = rest.QuoteID, rest.ExpiresAt, rest.QuoteTTLSeconds, rest.Task
	return nil
}

func (q TaskQuoteResponse) MarshalJSON() ([]byte, error) {
	out := map[string]any{}
	pricing, err := json.Marshal(q.PricingBreakdown)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(pricing, &out); err != nil {
		return nil, err
	}
	out["quoteId"], out["expiresAt"], out["quoteTtlSeconds"], out["task"] = q.QuoteID, q.ExpiresAt, q.QuoteTTLSeconds, q.Task
	return json.Marshal(out)
}

type CreateTaskResponse struct {
	Message     string           `json:"message"`
	Task        Task             `json:"task"`
	Pricing     PricingBreakdown `json:"pricing"`
	Idempotency map[string]any   `json:"idempotency"`
	Wallet      map[string]any   `json:"wallet,omitempty"`
	Payment     map[string]any   `json:"payment,omitempty"`
	Simulation  map[string]any   `json:"simulation,omitempty"`
}

type ListTasksParams struct {
	Limit  int
	Cursor string
	// Order is "asc" or "desc" (default).
	Order string
}

type TaskListItem struct {
	ID                     string     `json:"id"`
	DeveloperID            string     `json:"developerId"`
	RiderID                *string    `json:"riderId"`
	ExternalID             *string    `json:"externalId"`
	Status                 TaskStatus `json:"status"`
	TaskType               TaskType   `json:"taskType"`
	Priority               string     `json:"priority"`
	AutoAssign             bool       `json:"autoAssign"`
	Stages                 int        `json:"stages"`
	DeliveryFee            int64      `json:"deliveryFee"`
	TotalFee               int64      `json:"totalFee"`
	CreatedAt              string     `json:"createdAt"`
	CompletionCodeRequired bool       `json:"completionCodeRequired"`
	CompletionCodeLength   *int       `json:"completionCodeLength"`
}

type ListTasksResponse struct {
	Data    []TaskListItem `json:"data"`
	HasMore bool           `json:"hasMore"`
	Cursor  *string        `json:"cursor"`
}

type CancelTaskRequest struct {
	Reason string `json:"reason,omitempty"`
}

type CancelTaskResponse struct {
	Message string     `json:"message"`
	TaskID  string     `json:"taskId"`
	Status  TaskStatus `json:"status"`
}

type ManualAssignRequest struct {
	// RiderIDs are riders from AvailableRiders to offer the task to; at most 5 are used.
	RiderIDs []string `json:"riderIds"`
	// OfferTimeoutSeconds is how long riders have to accept (15–300, default 60).
	OfferTimeoutSeconds int `json:"offerTimeoutSeconds,omitempty"`
}

type RejectedRider struct {
	RiderID string `json:"riderId"`
	Reason  string `json:"reason"`
}

type ManualAssignResponse struct {
	Message            string `json:"message"`
	TaskID             string `json:"taskId"`
	Status             string `json:"status"`
	WorkflowInstanceID string `json:"workflowInstanceId"`
	Assignment         struct {
		OfferID             string          `json:"offerId"`
		SubmittedRiderIDs   []string        `json:"submittedRiderIds"`
		AcceptedRiderIDs    []string        `json:"acceptedRiderIds"`
		RejectedRiders      []RejectedRider `json:"rejectedRiders"`
		RequestedRiderCount int             `json:"requestedRiderCount"`
		AcceptedRiderCount  int             `json:"acceptedRiderCount"`
		MaxRiders           int             `json:"maxRiders"`
		Truncated           bool            `json:"truncated"`
		OfferTimeoutSeconds int             `json:"offerTimeoutSeconds"`
	} `json:"assignment"`
}

type AvailableRidersParams struct {
	// Tier is 1, 2 or 3 (default 1): how far out to look.
	Tier int
}

type AvailableRider struct {
	RiderID    string         `json:"riderId"`
	FirstName  string         `json:"firstName"`
	DistanceKm float64        `json:"distanceKm"`
	Vehicle    map[string]any `json:"vehicle"`
	Rating     *RatingSummary `json:"rating,omitempty"`
}

type AvailableRidersResponse struct {
	TaskID              string           `json:"taskId"`
	Tier                int              `json:"tier"`
	MaxAssignableRiders int              `json:"maxAssignableRiders"`
	Riders              []AvailableRider `json:"riders"`
}

type RatingRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment,omitempty"`
}

type SubmitRatingResponse struct {
	RatingID string `json:"ratingId"`
	Rating   int    `json:"rating"`
	Comment  string `json:"comment,omitempty"`
}

type DisputeRequest struct {
	// Reason is NOT_DELIVERED, DAMAGED_ITEMS, WRONG_LOCATION, INCOMPLETE_DELIVERY, POOR_SERVICE or OTHER.
	Reason      string   `json:"reason"`
	Description string   `json:"description"`
	Evidence    []string `json:"evidence,omitempty"`
}

type DisputeTaskResponse struct {
	Message         string `json:"message"`
	DisputeID       string `json:"disputeId"`
	Status          string `json:"status"`
	Note            string `json:"note"`
	ReviewTimeframe string `json:"reviewTimeframe"`
}

type CreateWebhookRequest struct {
	URL         string   `json:"url"`
	Events      []string `json:"events"`
	Description string   `json:"description,omitempty"`
	// Mode is "live" (default) or "test", and must match your API key.
	Mode string `json:"mode,omitempty"`
}

type UpdateWebhookRequest struct {
	URL         string   `json:"url,omitempty"`
	Events      []string `json:"events,omitempty"`
	Description string   `json:"description,omitempty"`
}

type Webhook struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Mode        string   `json:"mode"`
	Description string   `json:"description,omitempty"`
	CreatedAt   string   `json:"createdAt"`
	Events      []string `json:"events"`
	Usage       struct {
		SuccessCount int     `json:"successCount"`
		ErrorCount   int     `json:"errorCount"`
		LastUsed     *string `json:"lastUsed,omitempty"`
	} `json:"usage"`
}

type CreateWebhookResponse struct {
	ID     string   `json:"id"`
	URL    string   `json:"url"`
	Mode   string   `json:"mode"`
	Events []string `json:"events"`
	// Secret signs every delivery. It is returned only once: store it.
	Secret string `json:"secret"`
}

type ListWebhooksResponse struct {
	Data    []Webhook `json:"data"`
	HasMore bool      `json:"hasMore"`
	Cursor  *string   `json:"cursor"`
}

type CreateWebSocketTokenRequest struct {
	// Scope lists what the token may track, as "task:<id>" (up to 5 IDs per entry, comma-separated).
	Scope []string `json:"scope"`
	// ExpiresIn is the token lifetime in seconds (60–7200, default 1800).
	ExpiresIn int `json:"expiresIn,omitempty"`
}

type WebSocketToken struct {
	Token     string   `json:"token"`
	ExpiresIn int      `json:"expiresIn"`
	ExpiresAt string   `json:"expiresAt"`
	Scope     []string `json:"scope"`
}

type PresenceParams struct {
	Latitude  float64
	Longitude float64
	// Resolution is BLOCK, NEIGHBOURHOOD (default) or DISTRICT.
	Resolution string
}

type PresencePoint struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type PresenceCell struct {
	CellID       string          `json:"cellId"`
	Resolution   string          `json:"resolution"`
	H3Resolution int             `json:"h3Resolution"`
	Center       PresencePoint   `json:"center"`
	Boundary     []PresencePoint `json:"boundary"`
	Density      string          `json:"density"`
	VehicleTypes []string        `json:"vehicleTypes"`
}

type PresenceResponse struct {
	Mode         string `json:"mode"`
	Simulated    bool   `json:"simulated"`
	Resolution   string `json:"resolution"`
	H3Resolution int    `json:"h3Resolution"`
	Area         struct {
		Center   PresencePoint `json:"center"`
		RadiusKm float64       `json:"radiusKm"`
	} `json:"area"`
	Cells  []PresenceCell `json:"cells"`
	Totals struct {
		AvailableRiders int            `json:"availableRiders"`
		ByVehicleType   map[string]int `json:"byVehicleType"`
	} `json:"totals"`
	MinRidersPerCell    int    `json:"minRidersPerCell"`
	Capped              bool   `json:"capped"`
	GeneratedAt         string `json:"generatedAt"`
	RefreshAfterSeconds int    `json:"refreshAfterSeconds"`
}
