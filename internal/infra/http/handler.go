package http

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

type HandlersDefinition []HandlerDefinition

type HandlerDefinition struct {
	Endpoint, Method string
	HandlerFunc      http.HandlerFunc
}

func NewHandler(definitions HandlersDefinition) http.Handler {
	r := chi.NewRouter()
	defaultMiddlewares := []func(handler http.Handler) http.Handler{middleware.DefaultLogger}
	middlewares := make([]func(http.Handler) http.Handler, 0, len(defaultMiddlewares))
	middlewares = append(middlewares, defaultMiddlewares...)
	for _, d := range definitions {
		r.With(middlewares...).MethodFunc(d.Method, d.Endpoint, d.HandlerFunc)
	}
	return r
}
