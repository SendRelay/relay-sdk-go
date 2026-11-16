package relay

// Errand builds errand stages. Each method returns the same Stage you could write by hand; they
// only save remembering which fields each type takes.
//
//	client.Tasks.Create(ctx, relay.CreateTaskRequest{
//		TaskType: relay.TaskTypeErrand,
//		Stages: []relay.Stage{
//			relay.Errand.Queue(bank, relay.ErrandStageOptions{Instructions: "Collect the hardware token", ExpectedMinutes: 30, MaxMinutes: 90}),
//			relay.Errand.Purchase(shoprite, []relay.PurchaseItem{{Name: "Peak milk 400g", Quantity: relay.Int(2)}}, relay.ErrandStageOptions{}),
//			relay.Errand.Dropoff(home, relay.ErrandStageOptions{Contact: &relay.StageContact{Phone: "+2348030000000"}}),
//		},
//	}, nil)
var Errand errandBuilders

type errandBuilders struct{}

// ErrandStageOptions is what every builder takes besides its required fields. Zero values are left out.
type ErrandStageOptions struct {
	Instructions    string
	Reference       string
	ExpectedMinutes int
	MaxMinutes      int
	OpeningHours    *OpeningHours
	Contact         *StageContact
	// DeliverItems (DROPOFF): the item or PURCHASE references to hand over here.
	DeliverItems []string
	// IfUnavailable (PURCHASE): SKIP, SIMILAR or ASK.
	IfUnavailable string
	// SpendLimit (PURCHASE and PAYMENT), in kobo.
	SpendLimit int64
	// PayWith (PURCHASE and PAYMENT): MERCHANT_TRANSFER or PREPAID.
	PayWith  string
	Merchant *MerchantAccount
	// BillReference (PAYMENT): a bill or account number.
	BillReference string
	// Questions (CUSTOM): optional questions for the rider to answer.
	Questions []ErrandQuestion
	// Simulate is for test keys only.
	Simulate *StageSimulate
}

func errandStage(t StageType, location Location, o ErrandStageOptions) Stage {
	s := Stage{
		Type:          t,
		Location:      location,
		Instructions:  o.Instructions,
		OpeningHours:  o.OpeningHours,
		Contact:       o.Contact,
		DeliverItems:  o.DeliverItems,
		IfUnavailable: o.IfUnavailable,
		PayWith:       o.PayWith,
		Merchant:      o.Merchant,
		Questions:     o.Questions,
		Simulate:      o.Simulate,
	}
	if o.Reference != "" {
		s.Reference = String(o.Reference)
	}
	if o.ExpectedMinutes > 0 {
		s.ExpectedMinutes = Int(o.ExpectedMinutes)
	}
	if o.MaxMinutes > 0 {
		s.MaxMinutes = Int(o.MaxMinutes)
	}
	if o.SpendLimit > 0 {
		s.SpendLimit = Int64(o.SpendLimit)
	}
	if o.BillReference != "" {
		s.BillReference = String(o.BillReference)
	}
	return s
}

// Pickup collects items to carry, as on any delivery.
func (errandBuilders) Pickup(location Location, items []Item, o ErrandStageOptions) Stage {
	s := errandStage(StageTypePickup, location, o)
	s.Items = items
	return s
}

// Dropoff hands over what was collected or bought. Without DeliverItems, everything comes here.
func (errandBuilders) Dropoff(location Location, o ErrandStageOptions) Stage {
	return errandStage(StageTypeDropoff, location, o)
}

// Queue waits in line or for a service, then reports back.
func (errandBuilders) Queue(location Location, o ErrandStageOptions) Stage {
	return errandStage(StageTypeQueue, location, o)
}

// Purchase buys a shopping list, within the prices you set.
func (errandBuilders) Purchase(location Location, items []PurchaseItem, o ErrandStageOptions) Stage {
	s := errandStage(StageTypePurchase, location, o)
	s.PurchaseItems = items
	return s
}

// Payment pays a bill or a fee and brings the receipt. amount is the bill you expect, in kobo.
func (errandBuilders) Payment(location Location, payee string, amount int64, o ErrandStageOptions) Stage {
	s := errandStage(StageTypePayment, location, o)
	s.Payee, s.Amount = String(payee), Int64(amount)
	return s
}

// Inspection looks at something and answers your questions.
func (errandBuilders) Inspection(location Location, questions []ErrandQuestion, o ErrandStageOptions) Stage {
	s := errandStage(StageTypeInspection, location, o)
	s.Questions = questions
	return s
}

// Custom is anything else, described in instructions. effort (S, M or L) sets the stage fee.
func (errandBuilders) Custom(location Location, instructions, effort string, o ErrandStageOptions) Stage {
	o.Instructions = instructions
	s := errandStage(StageTypeCustom, location, o)
	s.Effort = effort
	return s
}

// String returns a pointer to v, for optional request fields.
func String(v string) *string { return &v }

// Int returns a pointer to v, for optional request fields.
func Int(v int) *int { return &v }

// Int64 returns a pointer to v, for optional amounts in kobo.
func Int64(v int64) *int64 { return &v }
