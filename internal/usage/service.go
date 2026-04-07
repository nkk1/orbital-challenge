package usage

import (
	"context"
	"sync"
	"time"
)

// Item is the domain representation of one usage entry. The transport layer
// (server/api) is responsible for converting this into the OpenAPI shape.
type Item struct {
	MessageID  int64
	Timestamp  time.Time
	ReportName string // empty if not applicable
	Credits    float64
}

// upstream is the subset of *Client behaviour the service depends on. Defining
// it as an interface keeps the service testable with a fake.
type upstream interface {
	FetchMessages(ctx context.Context) ([]Message, error)
	FetchReport(ctx context.Context, id int64) (*Report, bool, error)
}

// Service computes usage for the current billing period.
type Service struct {
	client         upstream
	reportFetchers int // max concurrent report fetches
}

// NewService wires the service to the given client.
func NewService(client upstream) *Service {
	return &Service{
		client:         client,
		reportFetchers: 8,
	}
}

// CurrentPeriod returns the usage items for the current billing period.
func (s *Service) CurrentPeriod(ctx context.Context) ([]Item, error) {
	messages, err := s.client.FetchMessages(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]Item, len(messages))
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.reportFetchers)

	for i, msg := range messages {
		i, msg := i, msg
		items[i] = Item{
			MessageID: msg.ID,
			Timestamp: msg.Timestamp,
		}

		if msg.ReportID == nil {
			items[i].Credits = CalculateTextCredits(msg.Text)
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			report, found, err := s.client.FetchReport(ctx, *msg.ReportID)
			if err == nil && found {
				items[i].ReportName = report.Name
				items[i].Credits = report.CreditCost
				return
			}
			// 404 or transient error → fall back to text-based calculation.
			items[i].Credits = CalculateTextCredits(msg.Text)
		}()
	}
	wg.Wait()

	return items, nil
}
