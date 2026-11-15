# Changelog

## 1.14.0 — 2026-11-15

- `Tasks.GetByExternalID(ctx, externalID)`: the task with the `ExternalID` you gave it at create, as `Get` returns it, or `nil` with no error if none has it (`GET /v1/sdk/tasks/by-external-id/{externalId}`). The reference is percent-encoded, so it can hold a slash.

## 1.13.0 — 2026-11-15

- Pooled delivery: `TaskTypePooledDelivery` and `Window` (`PooledWindowRequest`) on `CreateTaskRequest` and `TaskQuoteRequest`. A quote without a window returns `Windows` (`[]PooledQuoteWindow`): each upcoming wave with its status, times, price range and `QuoteID`. Pooled tasks carry `Window` (`*TaskPooledWindow`) and `PricingBreakdown` gains `Kind`, `MaxTotal`, `MinTotal`, `FinalTotal` and `FinalAt`. Other task types are unchanged. See [pooled delivery](https://docs.sendrelay.com.ng/concepts/pooled-delivery).
- `SimulationOutcome` `POOLING_SOLO` for test pooled deliveries, and the `task.pooling.grouped` and `task.pooling.solo` webhook events.
- `TaskTypeBulkDelivery` is deprecated: create and quote refuse it (`TASK_TYPE_RENAMED`). Older tasks keep it.

## 1.12.0 — 2026-11-14

- The API no longer accepts `BIKE` as a vehicle type; send `MOTORCYCLE` in `VehicleType`. A request with `BIKE` now gets a 400, and the presence map no longer returns a `BIKE` count. No code change in the SDK.

## 1.11.0 — 2026-11-09

- `Requirements` on `Task`, `TaskQuoteResponse` and `PresenceResponse` (`TaskRequirements`): each vehicle SendRelay looks for and the equipment a rider on it needs, for example a box on a motorcycle for food. See [the box a task needs](https://docs.sendrelay.com.ng/concepts/dispatch#the-box-a-task-needs).
- `PresenceParams` takes a job: `TaskType`, `VehicleType`, `HasBox`, `HasRefrigeration`, `HasAirConditioning` (`*bool`, nil leaves them out), `EstimatedWeight` and `EstimatedSize`, with the quote's meanings. `Presence.Get` then counts only riders who could take that job. Without `TaskType` the map is unchanged.

## 1.10.0 — 2026-11-08

- `Events.List` and `Events.ListAll`: the event log, every event for your account and mode, kept for 30 days. `Types` filters by event type.
- `Tasks.Events(ctx, taskID)`: every event about one task, oldest first.
- `Claims`: insurance claims on a task, for insured goods lost, stolen or damaged after collection. `Get`, `CreateUpload` (a URL to PUT one photo or PDF to), `File`, `Respond` and `Withdraw`. Live keys only.
- `Webhooks.Redeliver(ctx, webhookID, RedeliverWebhookRequest{EventID: ...})`: send a logged event to an endpoint again, with the same id.
- `UpdatedSince` on `ListTasksParams`: tasks changed since a time, in `UpdatedAt` order, for reconciliation jobs.
- `Revision` and `Currency` on `Task` and `TaskListItem`. Task events carry `data.revision`: apply an event only if it is higher than the revision you stored.
- `Tasks.Update` and `Tasks.TrackingLink`. `Stage` gains `Contact`, `DeliverItems`, `CompletionCode`, `CompletionCodeLength`, `ArrivedAt` and `WaitSeconds`; `DispatchRequest` gains `ReadyAt` and `OnRiderCancel`; `CreateTaskRequest` and `Task` gain `RequirePhoto`.

## 1.9.0 — 2026-11-06

- Insurance on tasks: `Insured` on `CreateTaskRequest` and `TaskQuoteRequest`. Leave it nil for the default, on; a pointer to false turns it off for that task.
- `InsuranceFee` and `InsuredValue` on `PricingBreakdown`. The premium is now its own line: VAT applies to it, and fare premiums and the platform fee no longer do.

## 1.8.0 — 2026-11-03

- Keep looking for a rider and scheduled pickups ([dispatch strategies](https://docs.sendrelay.com.ng/tasks/keep-looking-and-scheduling)).
- `Dispatch *DispatchRequest` on `CreateTaskRequest` and `TaskQuoteRequest` (`DispatchImmediate`, `DispatchWaitForRider`, `DispatchScheduled`), and `Task.Dispatch` with the search window. `TaskStatusScheduled` and `TaskStatusSearching`.
- `Tasks.SimulateAdvance(ctx, taskID)` (test keys) starts a booking's search now.
- Task requests accept an optional `Item.Reference`, and optional `Location.CountryCode` (ISO 3166-1 alpha-2) and `Location.PostalCode`. They are returned on the task when sent, and not used in pricing yet.
- Statuses and webhook event types this version doesn't list arrive as their raw strings. SendRelay will add task statuses (for example for tasks that keep searching for a rider), so treat a status you don't recognise as still in progress.

## 1.7.0 — 2026-10-29

- First release, versioned with the other SendRelay server SDKs (they move together). Matches the shared SDK conformance suite (packages/sdks/conformance).
