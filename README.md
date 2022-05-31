# japi is a JSON HTTP API go library

Japi is a fast & simple HTTP API library that will automatically marshal JSON payloads to/from 
your request and response structs. It follows [RFC7807](https://datatracker.ietf.org/doc/html/rfc7807) 
standard for returning useful problem details.

This library focuses on happy path to minimize code and dependencies. For more complex use cases, 
we recommend sticking to a larger web framework. However, this library supports the standard 
net/http ecosystem.

This library requires Go 1.18 to work as it utilizes generics.

This library was forked from https://github.com/AbeMedia/go-don

## Contents

- [Basic Example](#basic-example)
- [Configuration](#configuration)
- [Request parsing](#request-parsing)
- [Customize response](#customize-response)
- [Problem details](#problem-details)
- [Sub-routers](#sub-routers)
- [Middleware](#middleware)

## Basic Example

```go
package main

import (
  "context"
  "errors"
  "fmt"
  "net/http"

  "github.com/jarrettv/go-japi"
)

type GreetRequest struct {
  Name string `path:"name"`         // Get name from the URL path.
  Age  int    `header:"X-User-Age"` // Get age from HTTP header.
}

type GreetResponse struct {
  // Remember to add tags for automatic marshalling
  Greeting string `json:"data"`
}

func Greet(ctx context.Context, req GreetRequest) (*GreetResponse, error) {
  if req.Name == "" {
    return nil, problem.Validation(map[string]string{
      "name": "required",
    })
  }
  res := &GreetResponse{
    Greeting: fmt.Sprintf("Hello %s, you're %d years old.", req.Name, req.Age),
  }

  return res, nil
}

func Pong(context.Context, japi.Empty) (string, error) {
  return "pong", nil
}

func main() {
  r := japi.New(nil)
  r.Get("/ping", japi.H(Pong)) // Handlers are wrapped with `japi.H`.
  r.Post("/greet/:name", japi.H(Greet))
  r.ListenAndServe(":8080")
}
```

## Configuration

Japi is configured by passing in the `Config` struct to `japi.New`. We recommend you setup `ProblemConfig` at a minimum.

```go
r := japi.New(&japi.Config{
  ProblemConfig: problem.ProblemConfig{
    ProblemTypeUrlFormat: "https://example.com/errors/%s",
    ProblemInstanceFunc: func(ctx context.Context) string {
      return fmt.Sprintf("https://example.com/trace/%d", time.Now().UnixMilli())
    },
  },
})
```
### RouteLogFunc

A function to easily log the route name and route variables.

### ProblemLogFunc

A function to easily log when problems occur.

### ProblemConfig.ProblemTypeUrlFormat

The format for the problem details type URI. See [RFC7807](https://datatracker.ietf.org/doc/html/rfc7807)

### ProblemConfig.ProblemInstanceFunc

A function for generating a unique trace URI. Defaults to a timestamp. See [RFC7807](https://datatracker.ietf.org/doc/html/rfc7807)

## Request parsing

Automatically unmarshals values from headers, URL query, URL path & request body into your request
struct.

```go
type MyRequest struct {
  // Get from the URL path.
  ID int64 `path:"id"`

  // Get from the URL query.
  Filter string `query:"filter"`

  // Get from the JSON or form body.
  Content float64 `form:"bar" json:"bar"`

  // Get from the HTTP header.
  Lang string `header:"Accept-Language"`
}
```

Please note that using a pointer as the request type negatively affects performance.

## Customize Response

Implement the `StatusCoder` and `Headerer` interfaces to customise headers and response codes.

```go
type MyResponse struct {
  Foo  string `json:"foo"`
}

// Set a custom HTTP response code.
func (nr *MyResponse) StatusCode() int {
  return 201
}

// Add custom headers to the response.
func (nr *MyResponse) Header() http.Header {
  header := http.Header{}
  header.Set("foo", "bar")
  return header
}
```

## Problems

Return a `problem.Problem` error when something goes wrong. For example:

```go
return nil, problem.Unexpected(err) // 500
// or
return nil, problem.NotFound() // 404
// or
return nil, problem.NotPermitted(username) // 403
// or
return nil, problem.Validation(map[string]string{ // 400
  "name": "required",
})
// or
return nil, problem.RuleViolantion("item is on backorder") // 400
// or
return nil, problem.NotCurrent() // 407
```


## Sub-routers

You can create sub-routers using the `Group` function:

```go
r := japi.New(nil)
sub := r.Group("/api")
sub.Get("/hello")
```

## Middleware

Japi uses the standard http middleware format of
`func(http.RequestHandler) http.RequestHandler`.

For example:

```go
func loggingMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request)  {
    log.Println(r.URL)
    next(ctx)
  })
}
```

It is registered on a router using `Use` e.g.

```go
r := japi.New(nil)
r.Post("/", japi.H(handler))
r.Use(loggingMiddleware)
```

Middleware registered on a group only applies to routes in that group and child groups.

```go
r := japi.New(nil)
r.Get("/login", japi.H(loginHandler))
r.Use(loggingMiddleware) // applied to all routes

api := r.Group("/api")
api.Get("/hello", japi.H(helloHandler))
api.Use(authMiddleware) // applied to routes `/api/hello` and `/api/v2/bye`


v2 := api.Group("/v2")
v2.Get("/bye", japi.H(byeHandler))
v2.Use(corsMiddleware) // only applied to `/api/v2/bye`

```

To pass values from the middleware to the handler extend the context e.g.

```go
func myMiddleware(next http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request)  {
    ctx := context.WithValue(r.Context(), ContextUserKey, "my_user")
    next.ServeHTTP(w, r.WithContext(ctx))
  })
}
```

This can now be accessed in the handler:

```go
user := ctx.Value(ContextUserKey).(string)
```
