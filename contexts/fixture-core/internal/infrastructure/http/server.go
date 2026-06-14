// Package http contiene el servidor HTTP y los handlers REST de fixture-core.
// Usa chi como router por su compatibilidad con net/http stdlib y su soporte
// nativo para middleware.
package http

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/wc-fixture/shared/pkg/logger"
)

// ServerConfig contiene los parámetros del servidor HTTP.
type ServerConfig struct {
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	IdleTimeout     time.Duration
	ShutdownTimeout time.Duration
}

// DefaultServerConfig retorna una configuración razonable para producción.
func DefaultServerConfig(port int) ServerConfig {
	return ServerConfig{
		Port:            port,
		ReadTimeout:     10 * time.Second,
		WriteTimeout:    30 * time.Second,
		IdleTimeout:     60 * time.Second,
		ShutdownTimeout: 15 * time.Second,
	}
}

// Server encapsula el servidor HTTP con su configuración y router.
type Server struct {
	cfg    ServerConfig
	router http.Handler
}

func NewServer(cfg ServerConfig, router http.Handler) *Server {
	return &Server{cfg: cfg, router: router}
}

// Start inicia el servidor HTTP y bloquea hasta que ctx sea cancelado.
// Al cancelarse ctx ejecuta un shutdown graceful esperando hasta ShutdownTimeout.
func (s *Server) Start(ctx context.Context) error {
	log := logger.FromContext(ctx)

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Port),
		Handler:      s.router,
		ReadTimeout:  s.cfg.ReadTimeout,
		WriteTimeout: s.cfg.WriteTimeout,
		IdleTimeout:  s.cfg.IdleTimeout,
	}

	// Iniciar en goroutine — el error de ListenAndServe (distinto de ErrServerClosed)
	// se comunica via el canal errCh.
	errCh := make(chan error, 1)
	go func() {
		log.Info("servidor HTTP iniciado", "port", s.cfg.Port)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case <-ctx.Done():
		// Shutdown graceful
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()

		log.Info("iniciando shutdown del servidor HTTP")
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("http_server: error en shutdown: %w", err)
		}
		log.Info("servidor HTTP detenido correctamente")
		return nil

	case err := <-errCh:
		return fmt.Errorf("http_server: error fatal: %w", err)
	}
}
