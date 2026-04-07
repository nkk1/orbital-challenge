package usage

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const (
	defaultMessagesURL = "https://owpublic.blob.core.windows.net/tech-task/messages/current-period"
	defaultReportURL   = "https://owpublic.blob.core.windows.net/tech-task/reports/%d"
)

// Message is a message returned by the upstream messages endpoint.
type Message struct {
	ID        int64     `json:"id"`
	Text      string    `json:"text"`
	Timestamp time.Time `json:"timestamp"`
	ReportID  *int64    `json:"report_id,omitempty"`
}

// Report is a report returned by the upstream reports endpoint.
type Report struct {
	ID         int64   `json:"id"`
	Name       string  `json:"name"`
	CreditCost float64 `json:"credit_cost"`
}

// Client talks to the upstream Orbital Copilot endpoints.
type Client struct {
	httpClient  *http.Client
	messagesURL string
	reportURL   string // expects %d for the report id
}

// NewClient returns a Client with sensible defaults.
func NewClient() *Client {
	return &Client{
		httpClient:  &http.Client{Timeout: 10 * time.Second},
		messagesURL: defaultMessagesURL,
		reportURL:   defaultReportURL,
	}
}

type messagesResponse struct {
	Messages []Message `json:"messages"`
}

// FetchMessages returns the messages for the current billing period.
func (c *Client) FetchMessages(ctx context.Context) ([]Message, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.messagesURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("messages endpoint returned %d", resp.StatusCode)
	}
	var mr messagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&mr); err != nil {
		return nil, err
	}
	return mr.Messages, nil
}

// FetchReport returns the report for the given id. If the upstream returns
// 404, FetchReport returns (nil, false, nil) so the caller can fall back.
func (c *Client) FetchReport(ctx context.Context, id int64) (*Report, bool, error) {
	url := fmt.Sprintf(c.reportURL, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, false, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, false, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("reports endpoint returned %d", resp.StatusCode)
	}
	var r Report
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return nil, false, err
	}
	return &r, true, nil
}
