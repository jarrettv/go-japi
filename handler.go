package japi

import (
	"context"
	"net/http"

	"github.com/goccy/go-json"

	"github.com/jarrettv/go-japi/decoder"
	"github.com/jarrettv/go-japi/problem"
	"github.com/julienschmidt/httprouter"
)

// StatusCoder allows you to customise the HTTP response code.
type StatusCoder interface {
	StatusCode() int
}

// Headerer allows you to customise the HTTP headers.
type Headerer interface {
	Header() http.Header
}

// Problemer allows you to customize the error problem details.
type Problemer interface {
	Problem() problem.Problem
}

// Handler allows you to handle request with the route params
type Handler interface {
	http.Handler
	handle(http.ResponseWriter, *http.Request, httprouter.Params)
}

// H wraps your handler function with the Go generics magic.
func H[T any, O any](handle Handle[T, O]) Handler {
	h := &handler[T, O]{handler: handle}

	var t T

	if hasTag(t, headerTag) {
		dec, err := decoder.NewCachedDecoder(t, headerTag)
		if err == nil {
			h.decodeHeader = dec
		}
	}

	if hasTag(t, queryTag) {
		dec, err := decoder.NewMapDecoder(t, queryTag)
		if err == nil {
			h.decodeQuery = dec
		}
	}

	if hasTag(t, pathTag) {
		dec, err := decoder.NewParamsDecoder(t, pathTag)
		if err == nil {
			h.decodePath = dec
		}
	}

	return h
}

// E creates a Handler that returns the error
func E(err error) Handler {
	return H(func(context.Context, *Empty) (*Empty, error) {
		return nil, err
	})
}

// LoadRawJson loads raw json to response useful for swagger docs.
func LoadRawJson(loadRawJson func() ([]byte, error)) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json, e := loadRawJson()
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
			return
		}
		RawJson(json)
	})
}

// RawJson sends raw json to response useful for swagger docs.
func RawJson(data []byte) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, e := w.Write(data)
		if e != nil {
			http.Error(w, e.Error(), http.StatusInternalServerError)
		}
	})
}

// Handle is the type for your handlers.
type Handle[T any, O any] func(ctx context.Context, request T) (O, error)

type handler[T any, O any] struct {
	config       *Config
	handler      Handle[T, O]
	decodeHeader *decoder.CachedDecoder
	decodePath   *decoder.ParamsDecoder
	decodeQuery  *decoder.MapDecoder
	isNil        func(v any) bool
}

func (h *handler[T, O]) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.handle(w, r, nil)
}

//nolint:gocognit,cyclop
func (h *handler[T, O]) handle(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if h.config.RouteLogFunc != nil {
		route := p.MatchedRoutePath()
		vars := make(map[string]string, len(p))
		for _, param := range p {
			if param.Value != route {
				vars[param.Key] = param.Value
			}
		}
		h.config.RouteLogFunc(r.Context(), route, vars) // TODO (jv) get route
	}

	serveProblem := func(p *problem.Problem) {
		h.config.Enrich(r.Context(), p)
		if h.config.ProblemLogFunc != nil {
			h.config.ProblemLogFunc(r.Context(), p)
		}
		p.ServeJSON(w)
	}

	serveRequestProblem := func(e error) {
		p := problem.BadRequest(e)
		serveProblem(p)
	}

	req := new(T)

	// Decode the header
	if h.decodeHeader != nil {
		e := h.decodeHeader.Decode(r.Header, req)
		if e != nil {
			serveRequestProblem(e)
			return
		}
	}

	// Decode the URL query
	if h.decodeQuery != nil && r.URL.RawQuery != "" {
		e := h.decodeQuery.Decode(r.URL.Query(), req)
		if e != nil {
			serveRequestProblem(e)
			return
		}
	}

	// Decode the path params
	if h.decodePath != nil && len(p) != 0 {
		e := h.decodePath.Decode(p, req)
		if e != nil {
			serveRequestProblem(e)
			return
		}
	}

	// Decode the body
	if r.ContentLength > 0 {
		dec, e := getDecoder(JsonEncoding)
		if e != nil {
			serveRequestProblem(e) // http.ErrNotSupported
			return
		}

		if e := dec(r, req); e != nil {
			serveRequestProblem(e)
			return
		}
	}

	var res any
	res, e := h.handler(r.Context(), *req)
	w.Header().Set("Content-Type", JsonEncoding+"; charset=utf-8")
	if e != nil {
		if pb, ok := e.(Problemer); ok {
			p := pb.Problem()
			serveProblem(&p)
			return
		} else if p, ok := e.(*problem.Problem); ok {
			serveProblem(p)
			return
		} else {
			p := problem.Unexpected(e)
			serveProblem(p)
			return
		}
	}

	if h, ok := res.(Headerer); ok {
		headers := w.Header()
		for k, v := range h.Header() {
			headers[k] = v
		}
	}

	if sc, ok := res.(StatusCoder); ok {
		w.WriteHeader(sc.StatusCode())
	}

	if e = json.NewEncoder(w).Encode(res); e != nil {
		p := problem.Unexpected(e)
		serveProblem(p)
	}
}

func (h *handler[T, O]) setConfig(r *Config) {
	h.config = r
}

const (
	headerTag = "header"
	pathTag   = "path"
	queryTag  = "query"
)
