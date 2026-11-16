package relay

import (
	"encoding/json"
	"testing"
)

func stageJSON(t *testing.T, v any) map[string]any {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]any{}
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func TestErrandBuildersMakeTheStagesYouCouldWriteByHand(t *testing.T) {
	shop := Location{Latitude: 6.4433, Longitude: 3.4796, Address: "Shoprite, Lekki"}
	got := stageJSON(t, Errand.Purchase(shop, []PurchaseItem{{Name: "Peak milk 400g", Quantity: Int(2), MaxUnitPrice: Int64(350000)}}, ErrandStageOptions{Reference: "groceries", PayWith: "PREPAID"}))
	if got["type"] != "PURCHASE" || got["reference"] != "groceries" || got["payWith"] != "PREPAID" {
		t.Fatalf("unexpected purchase %v", got)
	}
	items := got["items"].([]any)
	first := items[0].(map[string]any)
	if first["name"] != "Peak milk 400g" || first["quantity"] != float64(2) || first["maxUnitPrice"] != float64(350000) {
		t.Fatalf("unexpected items %v", items)
	}
	if _, ok := first["estimatedValue"]; ok {
		t.Fatal("a PURCHASE item must not carry delivery item fields")
	}

	payment := stageJSON(t, Errand.Payment(shop, "Eko Electricity", 1500000, ErrandStageOptions{BillReference: "0123456789"}))
	if payment["payee"] != "Eko Electricity" || payment["amount"] != float64(1500000) || payment["billReference"] != "0123456789" {
		t.Fatalf("unexpected payment %v", payment)
	}
	custom := stageJSON(t, Errand.Custom(shop, "Hand the keys to the gateman", "S", ErrandStageOptions{}))
	if custom["effort"] != "S" || custom["instructions"] != "Hand the keys to the gateman" {
		t.Fatalf("unexpected custom %v", custom)
	}
	queue := stageJSON(t, Errand.Queue(shop, ErrandStageOptions{ExpectedMinutes: 30}))
	if len(queue) != 3 || queue["expectedMinutes"] != float64(30) {
		t.Fatalf("a queue stage carries only what was set: %v", queue)
	}
}

func TestPurchaseItemsRoundTrip(t *testing.T) {
	raw := `{"type":"PURCHASE","location":{"latitude":1,"longitude":2,"address":"a"},"items":[{"name":"Milk","quantity":2,"ifUnavailable":"ASK"}]}`
	var s Stage
	if err := json.Unmarshal([]byte(raw), &s); err != nil {
		t.Fatal(err)
	}
	if len(s.PurchaseItems) != 1 || s.PurchaseItems[0].IfUnavailable != "ASK" || len(s.Items) != 0 {
		t.Fatalf("purchase items not read: %+v", s)
	}
	out, _ := json.Marshal(s)
	var back, want any
	_ = json.Unmarshal(out, &back)
	_ = json.Unmarshal([]byte(raw), &want)
	a, _ := json.Marshal(back)
	b, _ := json.Marshal(want)
	if string(a) != string(b) {
		t.Fatalf("round trip changed the stage:\n%s\n%s", a, b)
	}
}
