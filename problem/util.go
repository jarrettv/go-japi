package problem

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ProblemConfig struct {
	// the URI to be string formatted with default type codes
	ProblemTypeUrlFormat string
	// the function to return URI for this problem instance
	ProblemInstanceFunc func(ctx context.Context) string
}

// Enrich will alter the type and instance of the problem as configured.
func (cfg ProblemConfig) Enrich(ctx context.Context, p *Problem) {
	if cfg.ProblemTypeUrlFormat != "" && p.Type != "about:blank" && !strings.HasPrefix(p.Type, "http") {
		p.Type = fmt.Sprintf(cfg.ProblemTypeUrlFormat, p.Type)
	}
	if cfg.ProblemInstanceFunc != nil {
		p.Instance = cfg.ProblemInstanceFunc(ctx)
	}
}

// ServeJSON will output Problem Details json to the response writer.
func (pd *Problem) ServeJSON(w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(pd.Status)
	if err := json.NewEncoder(w).Encode(pd); err != nil {
		return err
	}
	return nil
}
