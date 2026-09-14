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
	"github.com/jjcheng/wawa-go/internal/setup"
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
	// extract the args
	task := "queue"
	if len(os.Args) > 1 && os.Args[1] != "" {
		task = os.Args[1]
	}
	switch task {
	case "queue":
		setup.StartQueueListener(ctx, dependencies)
	default:
		log.Printf("unknown worker task: %s\n", task)
	}
	log.Println("worker stopped")
	log.Println("server exiting")
}
