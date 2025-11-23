package main

import (
	"context"
	"fmt"
	"io"
	"koditon-go/internal/config"
	"koditon-go/internal/server"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Config struct {
	Host            string
	Port            string
	ShutdownTimeout time.Duration
}

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout, stderr io.Writer,
) error {
	ctx, cancel := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer cancel()

	cfg, err := config.Load(args, getenv, stderr)
	if err != nil {
		return err
	}

	logger := log.New(stdout, "app ", log.LstdFlags|log.LUTC)

	handler := server.New(logger, cfg)
	httpServer := &http.Server{
		Addr:    net.JoinHostPort(cfg.Host, cfg.Port),
		Handler: handler,
		BaseContext: func(net.Listener) context.Context {
			return ctx
		},
	}

	errCh := make(chan error, 1)
	go func() {
		logger.Printf("listening on %s", httpServer.Addr)
		if serveErr := httpServer.ListenAndServe(); serveErr != nil && serveErr != http.ErrServerClosed {
			errCh <- serveErr
			return
		}
		errCh <- nil
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer shutdownCancel()
		logger.Printf("shutting down (timeout %s)", cfg.ShutdownTimeout)
		if shutdownErr := httpServer.Shutdown(shutdownCtx); shutdownErr != nil {
			return fmt.Errorf("server shutdown: %w", shutdownErr)
		}
		return nil
	case serveErr := <-errCh:
		if serveErr != nil {
			return fmt.Errorf("http server: %w", serveErr)
		}
		return nil
	}
}

type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}
