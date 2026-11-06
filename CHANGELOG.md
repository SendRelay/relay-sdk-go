# Changelog

## 1.9.0 — 2026-11-06

- Insurance on tasks: `Insured` on `CreateTaskRequest` and `TaskQuoteRequest`. Leave it nil for the default, on; a pointer to false turns it off for that task.
- `InsuranceFee` and `InsuredValue` on `PricingBreakdown`. The premium is now its own line: VAT applies to it, and fare premiums and the platform fee no longer do.

## 1.8.0 — 2026-11-03

- Keep looking for a rider and scheduled pickups ([dispatch strategies](https://docs.sendrelay.com.ng/tasks/keep-looking-and-scheduling)).
- `Dispatch *DispatchRequest` on `CreateTaskRequest` and `TaskQuoteRequest` (`DispatchImmediate`, `DispatchWaitForRider`, `DispatchScheduled`), and `Task.Dispatch` with the search window. `TaskStatusScheduled` and `TaskStatusSearching`.
- `Tasks.SimulateAdvance(ctx, taskID)` (test keys) starts a booking's search now.
- Task requests accept an optional `Item.Reference`, and optional `Location.CountryCode` (ISO 3166-1 alpha-2) and `Location.PostalCode`. They are returned on the task when sent, and not used in pricing yet.
- Statuses and webhook event types this version doesn't list arrive as their raw strings. Relay will add task statuses (for example for tasks that keep searching for a rider), so treat a status you don't recognise as still in progress.

## 1.7.0 — 2026-10-29

- First release, versioned with the other Relay server SDKs (they move together). Matches the shared SDK conformance suite (packages/sdks/conformance).
