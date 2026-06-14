package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/wc-fixture/shared/pkg/logger"
)

type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

func DefaultServerConfig(port int) ServerConfig {
	return ServerConfig{
		Port:            port,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 15 * time.Second,
	}
}

type Server struct {
	cfg    ServerConfig
	router http.Handler
}

func NewServer(cfg ServerConfig, router http.Handler) *Server {
	return &Server{cfg: cfg, router: router}
}

func (s *Server) Start(ctx context.Context) error {
	log := logger.FromContext(ctx)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Port),
		Handler:      s.router,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		IdleTimeout:  s.cfg.IdleTimeout,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Info("servidor HTTP team-registry iniciado", "port", s.cfg.Port)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("team_registry/http: error en shutdown: %w", err)
		}
		log.Info("servidor HTTP team-registry detenido")
		return nil
	case err := <-errCh:
		return fmt.Errorf("team_registry/http: error fatal: %w", err)
	}
}
