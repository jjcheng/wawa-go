package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/setup"
)

func main() {
	// set timezone to utc so no need to call .UTC() everytime
	time.Local = time.UTC
	// set log to stdout as error will be handled by loggerService
	log.SetOutput(os.Stdout)
	log.Println("starting worker")
	log.Printf("environment: %s\n", cfg.Default().Site.Environment)
	// setup database
	logger := service.NewLogger()
	unitOfWork, err := setup.SetupDatabase(cfg.Default().Database.DSN(), logger)
	if err != nil {
		log.Fatalf("failed to setup database: %v", err)
	}
	dependencies := setup.SetupServices(unitOfWork, logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	log.Println("worker started")
	setup.StartQueueListener(ctx, dependencies)
	log.Println("worker stopped")
	log.Println("zeroize global keys")
	cfg.Default().Site.GlobalKeys.Zeroize()
	log.Println("shutting down telemetry service")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if telemetryService := logger.TelemetryService(); telemetryService != nil {
		if err := telemetryService.Shutdown(shutdownCtx); err != nil {
			logger.ErrorFunction(err, "worker telemetry failed to shutdown")
		}
	}
	log.Println("server exiting")
}
