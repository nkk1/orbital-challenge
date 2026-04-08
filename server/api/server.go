package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/nkk1/orbital-challenge/internal/usage"
)

// Server implements the generated ServerInterface from api.gen.go.
type Server struct {
	usage *usage.Service
}

// NewServer constructs a Server backed by the given usage service.
func NewServer(svc *usage.Service) *Server {
	return &Server{usage: svc}
}

// GetUsage handles GET /usage. The method name and signature must match the
// ServerInterface produced by oapi-codegen for operationId getUsage.
func (s *Server) GetUsage(w http.ResponseWriter, r *http.Request) {
	items, err := s.usage.CurrentPeriod(r.Context())
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to fetch messages: "+err.Error())
		return
	}

	resp := UsageResponse{
		Usage: toAPIItems(items),
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("encode error: %v", err)
	}
}

// toAPIItems converts domain items into the generated OpenAPI types.
func toAPIItems(items []usage.Item) []UsageItem {
	out := make([]UsageItem, len(items))
	for i, it := range items {
		out[i] = UsageItem{
			MessageId: it.MessageID,
			Timestamp: it.Timestamp,
			Credits:   it.Credits,
		}
		if it.ReportName != "" {
			name := it.ReportName
			out[i].ReportName = &name
		}
	}
	return out
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(Error{Error: msg})
}
