package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/jarrettv/go-japi"
	"github.com/jarrettv/go-japi/problem"
)

func Empty(context.Context, *japi.Empty) (interface{}, error) {
	return nil, nil
}

func Ping(context.Context, *japi.Empty) (string, error) {
	return "pong", nil
}

type GreetRequest struct {
	Name string `json:"name"`         // Get name from JSON body
	Age  int    `header:"X-User-Age"` // Get age from HTTP header
}

type GreetResponse struct {
	Greeting string `json:"data"`
}

type HelloRequest struct {
	First string `path:"first"` // Get name from JSON body
	Last  string `path:"last"`  // Get name from JSON body
}

type HelloResponse struct {
	Hello string `json:"hello"`
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
		return nil, problem.Validation(map[string]string{
			"name": "required",
		})
	}
	res := &GreetResponse{
		Greeting: fmt.Sprintf("Hello %s, you're %d years old.", req.Name, req.Age),
	}
	return res, nil
}

func Hello(ctx context.Context, req *HelloRequest) (*HelloResponse, error) {
	res := &HelloResponse{
		Hello: fmt.Sprintf("Hello %s %s", req.First, req.Last),
	}
	return res, nil
}

func main() {
	r := japi.New(&japi.Config{
		ProblemConfig: problem.ProblemConfig{
			ProblemTypeUrlFormat: "https://example.com/errors/%s",
			ProblemInstanceFunc: func(ctx context.Context) string {
				return fmt.Sprintf("https://example.com/trace/%d", time.Now().UnixMilli())
			},
		},
	})
	r.Get("/", japi.H(Empty))

	g := r.Group("/api")
	g.Get("/ping", japi.H(Ping))
	g.Post("/greet", japi.H(Greet))
	g.Post("/hello/:first/:last", japi.H(Hello))
	http.ListenAndServe(":8080", r.Router())
}
