//go:build smoke

// Package api smoke tests run the real handler against the real upstream
// Orbital endpoints. They are gated behind the `smoke` build tag because
// they make live network calls and depend on the upstream payload not
// drifting.
//
// Run with:
//
//	go test -tags smoke -v ./server/api/...
package api_test

import (
	"encoding/json"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/nkk1/orbital-challenge/internal/usage"
	"github.com/nkk1/orbital-challenge/server/api"
)

const upstreamMessagesURL = "https://owpublic.blob.core.windows.net/tech-task/messages/current-period"

// approxEq tolerates the floating-point drift accumulated by the credit
// calculation's 0.05 / 0.1 / 0.2 / 0.3 increments.
func approxEq(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

// upstreamReachable checks the messages endpoint with a short timeout. If
// the upstream is down or the network is unavailable, the smoke test skips
// rather than failing — this file is a smoke test, not a unit test.
func upstreamReachable(t *testing.T) {
	t.Helper()
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(upstreamMessagesURL)
	if err != nil {
		t.Skipf("upstream unreachable, skipping smoke test: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("upstream returned %d, skipping smoke test", resp.StatusCode)
	}
}

// upstreamMessageCount fetches the live messages payload and returns its
// length, used for the count-matches structural check.
func upstreamMessageCount(t *testing.T) int {
	t.Helper()
	resp, err := http.Get(upstreamMessagesURL)
	if err != nil {
		t.Fatalf("fetch upstream messages: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read upstream messages: %v", err)
	}
	var payload struct {
		Messages []json.RawMessage `json:"messages"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		t.Fatalf("unmarshal upstream messages: %v", err)
	}
	return len(payload.Messages)
}

// startServer wires the real client (pointing at the real upstream) into the
// real handler and exposes it via httptest.NewServer.
func startServer(t *testing.T) *httptest.Server {
	t.Helper()
	svc := usage.NewService(usage.NewClient())
	apiServer := api.NewServer(svc)
	mux := http.NewServeMux()
	api.HandlerFromMux(apiServer, mux)
	return httptest.NewServer(mux)
}

// fetchUsage hits the local /usage and returns the items as a slice of
// loose maps so we can check whether keys are present-vs-absent (which a
// strongly-typed decode would hide).
func fetchUsage(t *testing.T, ts *httptest.Server) []map[string]any {
	t.Helper()
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(ts.URL + "/usage")
	if err != nil {
		t.Fatalf("GET /usage: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("status = %d: %s", resp.StatusCode, body)
	}
	var body struct {
		Usage []map[string]any `json:"usage"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return body.Usage
}

// itemByID locates an item in the response by its message_id. JSON numbers
// decode to float64 in map[string]any, so the comparison is on float64.
func itemByID(t *testing.T, items []map[string]any, id float64) map[string]any {
	t.Helper()
	for _, item := range items {
		if mid, ok := item["message_id"].(float64); ok && mid == id {
			return item
		}
	}
	t.Fatalf("no item with message_id %v in response", id)
	return nil
}

// TestSmoke_Usage runs the full slate against the real upstream.
func TestSmoke_Usage(t *testing.T) {
	upstreamReachable(t)

	ts := startServer(t)
	defer ts.Close()

	items := fetchUsage(t, ts)

	// ----- Structural checks: survive any upstream data drift. -----

	t.Run("response is non-empty", func(t *testing.T) {
		if len(items) == 0 {
			t.Fatal("got empty usage array")
		}
	})

	t.Run("item count matches upstream", func(t *testing.T) {
		want := upstreamMessageCount(t)
		if len(items) != want {
			t.Errorf("got %d items, upstream has %d messages", len(items), want)
		}
	})

	t.Run("every item is well-formed", func(t *testing.T) {
		for _, item := range items {
			id := item["message_id"]

			if _, ok := item["message_id"].(float64); !ok {
				t.Errorf("item %v: message_id missing or not numeric", id)
			}
			if _, ok := item["timestamp"].(string); !ok {
				t.Errorf("item %v: timestamp missing or not string", id)
			}
			credits, ok := item["credits"].(float64)
			if !ok {
				t.Errorf("item %v: credits missing or not numeric", id)
				continue
			}
			if credits < 1 {
				t.Errorf("item %v: credits=%v violates minimum-cost rule", id, credits)
			}
		}
	})

	t.Run("report_name when present is a non-empty string", func(t *testing.T) {
		for _, item := range items {
			raw, present := item["report_name"]
			if !present {
				continue
			}
			name, ok := raw.(string)
			if !ok {
				t.Errorf("item %v: report_name is not a string: %v", item["message_id"], raw)
				continue
			}
			if name == "" {
				t.Errorf("item %v: report_name is empty string", item["message_id"])
			}
		}
	})

	t.Run("timestamps parse as RFC3339", func(t *testing.T) {
		for _, item := range items {
			ts, _ := item["timestamp"].(string)
			if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
				t.Errorf("item %v: timestamp %q does not parse: %v",
					item["message_id"], ts, err)
			}
		}
	})

	// ----- Known-message assertions: depend on the live payload as of
	// the original task description. If the upstream data ever changes,
	// these will need to be re-derived. -----

	t.Run("id=1009 uses report 8806 (Maintenance Responsibilities Report, 94 credits)", func(t *testing.T) {
		// report_id 8806 → name and credit_cost from /reports/8806.
		// The text on the message is ignored entirely.
		item := itemByID(t, items, 1009)
		if got := item["report_name"]; got != "Maintenance Responsibilities Report" {
			t.Errorf("report_name = %v, want Maintenance Responsibilities Report", got)
		}
		if got := item["credits"].(float64); !approxEq(got, 94) {
			t.Errorf("credits = %v, want 94", got)
		}
	})

	t.Run("id=1104 text 'orbital latibro' → 2.10 credits (palindrome)", func(t *testing.T) {
		// "orbital latibro" with non-alphanumerics stripped is "orbitallatibro",
		// which reads the same forwards and backwards — it's a palindrome.
		// base 1 + 0.05*15 = 1.75
		// words: orbital(7,4-7) +0.2, latibro(7,4-7) +0.2 → +0.4
		// third vowels at indices 2,5,8,11,14 = 'b','a','l','i','o' → 'a','i','o' → 3 × 0.3 = +0.9
		// not >100, 2 unique → -2
		// subtotal: 1 + 0.75 + 0.4 + 0.9 - 2 = 1.05 (no clamp)
		// palindrome → *2 = 2.10
		item := itemByID(t, items, 1104)
		if _, present := item["report_name"]; present {
			t.Errorf("expected no report_name on text-only message, got %v", item["report_name"])
		}
		if got := item["credits"].(float64); !approxEq(got, 2.10) {
			t.Errorf("credits = %v, want 2.10", got)
		}
	})

	t.Run("id=1056 text 'What is the lease term?' → 1.85 credits", func(t *testing.T) {
		// base 1 + 0.05*23 = 2.15
		// words: What(4)+0.2, is(2)+0.1, the(3)+0.1, lease(5)+0.2, term(4)+0.2 → +0.8
		// third vowels at indices 2,5,8,11,14,17,20 = 'a','i','h','l','a','t','m' → 'a','i','a' → 3 × 0.3 = +0.9
		// not >100, 5 unique → -2
		// total: 1 + 1.15 + 0.8 + 0.9 - 2 = 1.85
		item := itemByID(t, items, 1056)
		if got := item["credits"].(float64); !approxEq(got, 1.85) {
			t.Errorf("credits = %v, want 1.85", got)
		}
	})

	t.Run("id=1076 text 'Tell me the name of the landlord.' has credits > 1", func(t *testing.T) {
		// Sanity check rather than an exact value — gives us one more
		// data point that the text path is producing real numbers.
		item := itemByID(t, items, 1076)
		if got := item["credits"].(float64); got <= 1 {
			t.Errorf("credits = %v, want > 1", got)
		}
	})
}
