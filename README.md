# Relay Go SDK

The official server SDK for [Relay](https://sendrelay.com.ng), the delivery API. Quote and create deliveries, read them back with the rider's live ETA, manage webhooks, and mint tracking tokens for your apps.

```bash
go get github.com/SendRelay/relay-sdk-go
```

Go 1.23 or newer. Standard library only.

## Quick start

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	relay "github.com/SendRelay/relay-sdk-go"
)

func main() {
	ctx := context.Background()
	client, err := relay.NewClient(os.Getenv("RELAY_API_KEY")) // sk_test_... or sk_live_..., server only
	if err != nil {
		log.Fatal(err)
	}

	stages := []relay.Stage{
		{
			Type:     relay.StageTypePickup,
			Location: relay.Location{Latitude: 6.5244, Longitude: 3.3792, Address: "12 Marina Road, Lagos"},
			Items:    []relay.Item{{Name: "Phone case", EstimatedValue: 500000, EstimatedWeight: "LIGHT", EstimatedSize: "BOX"}},
		},
		{Type: relay.StageTypeDropoff, Location: relay.Location{Latitude: 6.4474, Longitude: 3.4723, Address: "22 Admiralty Way, Lekki"}},
	}

	quote, err := client.Tasks.Quote(ctx, relay.TaskQuoteRequest{TaskType: relay.TaskTypePackageDelivery, Stages: stages})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("total (kobo):", quote.Total)

	created, err := client.Tasks.Create(ctx, relay.CreateTaskRequest{
		TaskType: relay.TaskTypePackageDelivery,
		Stages:   stages,
		QuoteID:  quote.QuoteID,
	}, &relay.CreateOptions{IdempotencyKey: "order-1042"}) // retries with the same key never create a second delivery
	if err != nil {
		log.Fatal(err)
	}

	read, err := client.Tasks.Get(ctx, created.Task.ID)
	if err != nil {
		log.Fatal(err)
	}
	if read.LiveEta != nil {
		fmt.Printf("arriving at stop %d at %s\n", read.LiveEta.StageIndex+1, read.LiveEta.ArrivesAt)
	}
}
```

## Every task

```go
for task, err := range client.Tasks.ListAll(ctx, &relay.ListTasksParams{Limit: 50}) {
	if err != nil {
		return err
	}
	fmt.Println(task.ID, task.Status)
}
```

## Webhooks

```go
body, _ := io.ReadAll(r.Body)
result := relay.VerifyWebhookSignature(body, r.Header.Get("X-Relay-Signature"), os.Getenv("RELAY_WEBHOOK_SECRET"), relay.DefaultWebhookTolerance)
if !result.Valid {
	http.Error(w, result.Error, http.StatusBadRequest)
	return
}
```

## Errors and retries

Use `errors.As` to tell errors apart:

- `*relay.APIError` has `StatusCode`, `Code` and `Details` from the API.
- `*relay.NetworkError` means the request never got an answer.
- `*relay.ValidationError` means the SDK rejected the input before sending.

Timeouts, 408, 429 and 5xx responses are retried up to 3 times with exponential backoff. Test-mode quota errors are not retried. Change this with `relay.WithMaxRetries(n)` or `relay.WithRetry(...)`.

Other options: `WithBaseURL`, `WithTimeout`, `WithHeaders`, `WithHTTPClient` and `WithoutAutoIdempotency`.

## Tracking in your app

```go
token, err := client.Auth.CreateWebSocketToken(ctx, relay.CreateWebSocketTokenRequest{Scope: []string{"task:" + taskID}})
```

Hand `token.Token` to your app, which tracks the task live with one of the UI SDKs.

[Full documentation](https://docs.sendrelay.com.ng/sdks/go-sdk)
