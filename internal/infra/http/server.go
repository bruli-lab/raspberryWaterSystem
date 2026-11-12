package http

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	corsx "github.com/rs/cors"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 30 * time.Second
	idleTimeout       = time.Minute
	shutdownTimeout   = 10 * time.Second
)

func newServer(ctx context.Context, handler http.Handler) *http.Server {
	requestsCtx := context.WithoutCancel(ctx)
	return &http.Server{
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
		BaseContext: func(_ net.Listener) context.Context {
			return requestsCtx
		},
	}
}

func listenAndServe(ctx context.Context, address string, server *http.Server, readyCh chan struct{}) error {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}
	close(readyCh)

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	err = server.Serve(ln)
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

type CORSOpt struct {
	AllowedOrigins     []string
	AllowedMethods     []string
	AllowedHeaders     []string
	OptionsPassthrough bool
}

func RunServer(ctx context.Context, serverURL string, httpHandler http.Handler, corsOpt *CORSOpt, logger *slog.Logger) error {
	logger.InfoContext(ctx, "[HTTP SERVICE] starting...")
	defer func() {
		logger.InfoContext(ctx, "[HTTP SERVICE] stopping...")
	}()

	readyCh := make(chan struct{})
	go func() {
		<-readyCh
		logger.InfoContext(ctx, fmt.Sprintf("[HTTP SERVICE] system ready to serve at %s", serverURL))
	}()

	corsHTTPHandler := buildCORS(corsOpt).Handler(httpHandler)

	server := newServer(ctx, corsHTTPHandler)

	return listenAndServe(ctx, serverURL, server, readyCh)
}

func buildCORS(corsOpt *CORSOpt) *corsx.Cors {
	var cors *corsx.Cors
	if corsOpt == nil {
		cors = corsx.AllowAll()
	} else {
		cors = corsx.New(corsx.Options{
			AllowedOrigins:     corsOpt.AllowedOrigins,
			AllowedMethods:     corsOpt.AllowedMethods,
			AllowedHeaders:     corsOpt.AllowedHeaders,
			OptionsPassthrough: corsOpt.OptionsPassthrough,
		})
	}
	return cors
}
