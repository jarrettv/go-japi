package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	min "github.com/jarrettv/go-minimal"
)

func Empty(context.Context, *min.Empty) (interface{}, error) {
	return nil, nil
}

func Ping(context.Context, *min.Empty) (string, error) {
	return "pong", nil
}

type GreetRequest struct {
	Name string `json:"name"`         // Get name from JSON body
	Age  int    `header:"X-User-Age"` // Get age from HTTP header
}

type GreetResponse struct {
	Greeting string `json:"data"`
}

func (gr *GreetResponse) StatusCode() int {
	return http.StatusTeapot
}

func (gr *GreetResponse) Header() http.Header {
	header := http.Header{}
	header.Set("foo", "bar")
	return header
}

func Greet(ctx context.Context, req *GreetRequest) (*GreetResponse, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, min.ProblemValid(map[string]string{
			"name": "required",
		})
	}
	res := &GreetResponse{
		Greeting: fmt.Sprintf("Hello %s, you're %d years old.", req.Name, req.Age),
	}
	return res, nil
}

func main() {
	r := min.New(&min.Config{
		ProblemTypeUrlFormat: "https://docs.example.com/errors/%s",
		ProblemInstanceFunc: func(_ context.Context) string {
			return fmt.Sprintf("https://errors.example.com/trace/%d", time.Now().UnixMilli())
		}})
	r.Get("/", min.H(Empty))

	g := r.Group("/api")
	g.Get("/ping", min.H(Ping))
	g.Post("/greet", min.H(Greet))

	http.ListenAndServe(":8080", r.Router())
}
