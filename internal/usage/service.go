package usage

import (
	"context"
	"sync"
	"time"
)

// represents each raw message in https://owpublic.blob.core.windows.net/tech-task/messages/current-period
type Item struct {
	MessageID  int64
	Timestamp  time.Time
	ReportName string // empty if not applicable
	Credits    float64
}

// mainly for mocking during unit-testing
type upstream interface {
	FetchMessages(ctx context.Context) ([]Message, error)
	FetchReport(ctx context.Context, id int64) (*Report, bool, error)
}

// Service computes usage for the current billing period.
type Service struct {
	// client to access the mock Oribital API endpoints
	client         upstream
	reportFetchers int // max concurrent report fetches
}

func NewService(client upstream) *Service {
	return &Service{
		client:         client,
		reportFetchers: 8,
	}
}

// CurrentPeriod fetch messages from Orbital API endpoints and convert them into 'Item's object
func (s *Service) CurrentPeriod(ctx context.Context) ([]Item, error) {
	messages, err := s.client.FetchMessages(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]Item, len(messages))
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.reportFetchers)

	for i, msg := range messages {
		items[i] = Item{
			MessageID: msg.ID,
			Timestamp: msg.Timestamp,
		}

		if msg.ReportID == nil {
			items[i].Credits = CalculateTextCredits(msg.Text)
			continue
		}

		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			report, found, err := s.client.FetchReport(ctx, *messages[i].ReportID)
			if err == nil && found {
				items[i].ReportName = report.Name
				items[i].Credits = report.CreditCost
				return
			}
			// 404 or transient error → fall back to text-based calculation.
			items[i].Credits = CalculateTextCredits(messages[i].Text)
		}(i)
	}
	wg.Wait()

	return items, nil
}
