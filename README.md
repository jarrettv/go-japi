# Minimal HTTP API go library

Minimal is a fast & simple HTTP API library. It will automatically marshal JSON payloads to/from 
your request and response structs. It follows [RFC7807](https://datatracker.ietf.org/doc/html/rfc7807) standard for 
returning useful problem details.

This library focuses on happy path to minimize code. For more complex use cases, we recommend sticking
to a larger web framework.

This library requires Go 1.18 to work as it utilizes generics.

This library was forked from https://github.com/AbeMedia/go-don

Let's keep this library minimal, please send PRs to delete unnecessary code.

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

  min "github.com/jarrettv/go-minimal"
)

type GreetRequest struct {
  Name string `path:"name"`         // Get name from the URL path.
  Age  int    `header:"X-User-Age"` // Get age from HTTP header.
}

type GreetResponse struct {
  // Remember to add all the tags for the renderers you enable.
  Greeting string `json:"data"`
}

func Greet(ctx context.Context, req GreetRequest) (*GreetResponse, error) {
  if req.Name == "" {
    return nil, min.ProblemValid(map[string]string{
      "name": "required",
    })
  }
  res := &GreetResponse{
    Greeting: fmt.Sprintf("Hello %s, you're %d years old.", req.Name, req.Age),
  }

  return res, nil
}

func Pong(context.Context, min.Empty) (string, error) {
  return "pong", nil
}

func main() {
  r := min.New(nil)
  r.Get("/ping", min.H(Pong)) // Handlers are wrapped with `minimal.H`.
  r.Post("/greet/:name", min.H(Greet))
  r.ListenAndServe(":8080")
}
```

## Configuration

Minimal is configured by passing in the `Config` struct to `minimal.New`.

```go
r := min.New(&min.Config{
  ProblemTypeUrlFormat: "https://docs.example.com/errors/%s",
  ProblemInstanceFunc: func(_ context.Context) string {
    return fmt.Sprintf("https://errors.example.com/trace/%d", time.Now().UnixMilli())
}})
```

### ProblemTypeUrlFormat

The format for the problem details type URI. See [RFC7807](https://datatracker.ietf.org/doc/html/rfc7807)

### ProblemInstanceFunc

A function for generating a unique trace URI. Defaults to a timestamp. See [RFC7807](https://datatracker.ietf.org/doc/html/rfc7807)

#### Form (input only)

MIME: `application/x-www-form-urlencoded`, `multipart/form-data`

Parses form data requests. Use the `form` tag in your request struct.

## Request parsing

Automatically unmarshals values from headers, URL query, URL path & request body into your request
struct.

```go
type MyRequest struct {
  // Get from the URL path.
  ID int64 `path:"id"`

  // Get from the URL query.
  Filter string `query:"filter"`

  // Get from the JSON, YAML, XML or form body.
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

## Problem Details

Implement the `Problemer` to customize your error problem details. Or use the many built in problems:

```go
return nil, min.ProblemUnexpected(e)
// or
return nil, min.ProblemNotFound()
// or
return nil, min.ProblemPermit(username)
// or
return nil, min.ProblemValid(map[string]string{
  "name": "required",
})
// or
return nil, min.ProblemRule("item is on backorder")
// or
return nil, min.ProblemOld()
```


## Sub-routers

You can create sub-routers using the `Group` function:

```go
r := min.New(nil)
sub := r.Group("/api")
sub.Get("/hello")
```

## Middleware

Minimal uses the standard http middleware format of
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
r := min.New(nil)
r.Post("/", min.H(handler))
r.Use(loggingMiddleware)
```

Middleware registered on a group only applies to routes in that group and child groups.

```go
r := min.New(nil)
r.Get("/login", min.H(loginHandler))
r.Use(loggingMiddleware) // applied to all routes

api := r.Group("/api")
api.Get("/hello", min.H(helloHandler))
api.Use(authMiddleware) // applied to routes `/api/hello` and `/api/v2/bye`


v2 := api.Group("/v2")
v2.Get("/bye", min.H(byeHandler))
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
