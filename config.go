package japi

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/goccy/go-json"

	"github.com/jarrettv/go-japi/problem"
)

type Config struct {
	// the function to call for logging route details
	RouteLogFunc func(ctx context.Context, route string, params map[string]string)
	// the function to call for logging problems
	ProblemLogFunc func(ctx context.Context, p *problem.Problem)
	problem.ProblemConfig
}

// GetDefaultConfig will return default problem config.
func GetDefaultConfig() *Config {
	return &Config{
		RouteLogFunc: func(ctx context.Context, route string, params map[string]string) {
			if params != nil && len(params) > 0 {
				data, err := json.Marshal(params)
				if err == nil {
					log.Printf("%s %s", route, string(data))
					return
				}
			}
			log.Print(route)
		},
		ProblemLogFunc: func(ctx context.Context, p *problem.Problem) {
			log.Printf("%v type=%v", p.Title, p.Type)
		},
		ProblemConfig: problem.ProblemConfig{
			ProblemTypeUrlFormat: "https://example.com/errors/%s",
			ProblemInstanceFunc: func(ctx context.Context) string {
				return fmt.Sprintf("https://example.com/trace/%d", time.Now().UnixMilli())
			},
		},
	}
}
