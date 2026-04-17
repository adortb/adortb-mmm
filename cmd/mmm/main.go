package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adortb/adortb-mmm/internal/api"
	"github.com/adortb/adortb-mmm/internal/metrics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	store := api.NewStore()
	handler := api.NewHandler(store)

	mux := http.NewServeMux()
	api.RegisterRoutes(mux, handler)
	mux.Handle("GET /metrics", metrics.MetricsHandler())
	mux.Handle("GET /health", metrics.HealthHandler())

	srv := &http.Server{
		Addr:         ":8105",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("MMM 服务启动", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("服务启动失败", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("服务优雅关闭失败", "error", err)
	}
	logger.Info("服务已停止")
}
