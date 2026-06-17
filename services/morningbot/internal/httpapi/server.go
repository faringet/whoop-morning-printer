package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/faringet/whoop-morning-printer/services/morningbot/config"
	"github.com/gin-gonic/gin"
)

type Server struct {
	log             *slog.Logger
	httpServer      *http.Server
	shutdownTimeout time.Duration
}

func NewServer(cfg config.HTTP, shutdownTimeout time.Duration, handler *Handler, auth *AuthMiddleware, log *slog.Logger) (*Server, error) {
	if !cfg.Enabled {
		return nil, errors.New("morningbot httpapi: http server is disabled")
	}
	if handler == nil {
		return nil, errors.New("morningbot httpapi: handler is nil")
	}
	if auth == nil {
		return nil, errors.New("morningbot httpapi: auth middleware is nil")
	}
	if log == nil {
		log = slog.Default()
	}
	if shutdownTimeout <= 0 {
		shutdownTimeout = 15 * time.Second
	}

	serverLog := log.With(
		slog.String("layer", "transport"),
		slog.String("module", "morningbot.httpapi.server"),
	)

	router := gin.New()
	router.HandleMethodNotAllowed = true

	if err := router.SetTrustedProxies(nil); err != nil {
		return nil, fmt.Errorf("morningbot httpapi: configure trusted proxies: %w", err)
	}

	router.Use(
		gin.Recovery(),
		requestLogger(serverLog),
		corsMiddleware(cfg.AllowedOrigins),
	)

	router.NoRoute(func(c *gin.Context) {
		writeError(c, http.StatusNotFound, "route_not_found", "Route was not found")
	})

	router.NoMethod(func(c *gin.Context) {
		writeError(c, http.StatusMethodNotAllowed, "method_not_allowed", "HTTP method is not allowed")
	})

	router.GET("/healthz", handler.Health)

	api := router.Group("/api/v1")
	api.Use(auth.RequireTelegramUser())
	api.GET("/wake-plan", handler.GetWakePlan)
	api.PUT("/wake-plan", handler.PutWakePlan)
	api.DELETE("/wake-plan", handler.DeleteWakePlan)

	httpServer := &http.Server{
		Addr:              cfg.Addr,
		Handler:           router,
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	return &Server{
		log:             serverLog,
		httpServer:      httpServer,
		shutdownTimeout: shutdownTimeout,
	}, nil
}

func (s *Server) Run(ctx context.Context) error {
	if s == nil || s.httpServer == nil {
		return errors.New("morningbot httpapi: server is nil")
	}

	errCh := make(chan error, 1)

	go func() {
		s.log.Info("http server started", slog.String("addr", s.httpServer.Addr))
		errCh <- s.httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("morningbot httpapi serve: %w", err)

	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		defer cancel()

		s.log.Info("http server stopping")

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("morningbot httpapi shutdown: %w", err)
		}

		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("morningbot httpapi serve after shutdown: %w", err)
		}

		s.log.Info("http server stopped")

		return ctx.Err()
	}
}

func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()

		c.Next()

		log.Info(
			"http request completed",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.Int("status", c.Writer.Status()),
			slog.Int("response_size", c.Writer.Size()),
			slog.Duration("duration", time.Since(startedAt)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}

func corsMiddleware(allowedOrigins []string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(allowedOrigins))

	for _, origin := range allowedOrigins {
		origin = strings.TrimSpace(origin)
		if origin != "" {
			allowed[origin] = struct{}{}
		}
	}

	return func(c *gin.Context) {
		origin := strings.TrimSpace(c.GetHeader("Origin"))

		if origin == "" || len(allowed) == 0 {
			c.Next()
			return
		}

		if _, ok := allowed[origin]; !ok {
			writeError(c, http.StatusForbidden, "origin_forbidden", "Request origin is not allowed")
			return
		}

		headers := c.Writer.Header()
		headers.Set("Access-Control-Allow-Origin", origin)
		headers.Set("Access-Control-Allow-Methods", "GET, PUT, DELETE, OPTIONS")
		headers.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Accept")
		headers.Set("Access-Control-Max-Age", "600")
		headers.Add("Vary", "Origin")
		headers.Add("Vary", "Access-Control-Request-Method")
		headers.Add("Vary", "Access-Control-Request-Headers")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
