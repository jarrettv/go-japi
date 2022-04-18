package minimal

import (
	"context"
	"github.com/goccy/go-json"
	"log"
	"net/http"

	"github.com/jarrettv/go-minimal/decoder"
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
	Problem() ProblemDetails
}

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

func E(err error) Handler {
	return H(func(context.Context, *Empty) (*Empty, error) {
		return nil, err
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
	serveProblem := func(status int) {
		p := ProblemStatus(status)
		p.enrich(r.Context(), h.config)
		p.serveJSON(w, r)
	}

	req := new(T)

	// Decode the header
	if h.decodeHeader != nil {
		e := h.decodeHeader.Decode(r.Header, req)
		if e != nil {
			serveProblem(http.StatusBadRequest)
			return
		}
	}

	// Decode the URL query
	if h.decodeQuery != nil && r.URL.RawQuery != "" {
		e := h.decodeQuery.Decode(r.URL.Query(), req)
		if e != nil {
			serveProblem(http.StatusBadRequest)
			return
		}
	}

	// Decode the path params
	if h.decodePath != nil && len(p) != 0 {
		e := h.decodePath.Decode(p, req)
		if e != nil {
			serveProblem(http.StatusBadRequest)
			return
		}
	}

	// Decode the body
	if r.ContentLength > 0 {
		dec, e := getDecoder(JsonEncoding)
		if e != nil {
			serveProblem(http.StatusNotAcceptable)
			return
		}

		if e := dec(r, req); e != nil {
			serveProblem(http.StatusBadRequest)
			return
		}
	}

	var res any
	res, e := h.handler(r.Context(), *req)
	w.Header().Set("Content-Type", JsonEncoding+"; charset=utf-8")
	if e != nil {
		if pb, ok := e.(Problemer); ok {
			p := pb.Problem()
			p.enrich(r.Context(), h.config)
			p.serveJSON(w, r)
		} else if p, ok := e.(*ProblemDetails); ok {
			p.enrich(r.Context(), h.config)
			p.serveJSON(w, r)
			return
		} else {
			p := ProblemUnexpected(e)
			p.enrich(r.Context(), h.config)
			p.serveJSON(w, r)
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
		log.Println(e) // TODO (jv) is this ok?
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
