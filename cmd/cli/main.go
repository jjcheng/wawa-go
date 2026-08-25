package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	dto_account "github.com/jjcheng/wawa-go/internal/dto/account"
	feature_cli "github.com/jjcheng/wawa-go/internal/feature/cli"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/setup"
)

var (
	_dependencies *service.Dependencies
	_user         dto_account.User
	_appId        int32
)

func main() {
	// set timezone to utc so no need to call .UTC() everytime
	_appId = 1
	time.Local = time.UTC
	logger := service.NewLogger()
	// setup database
	uow, err := setup.SetupDatabase(cfg.Default().Database.DSN(), logger)
	if err != nil {
		panic(fmt.Sprintf("FAILED to connect to DB: %v", err.Error()))
	}
	// setup services
	_dependencies = setup.SetupServices(uow, logger, false)
	log.Println("server started")
	fmt.Println()
	// AD-HOC: put any ad-hoc tasks here
	// crawl("https://www.gofit-gym.com/sg/")
	// extractKnowledge("https://www.gofit-gym.com/sg")
	// return

	// Setup graceful shutdown for interruption signals
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	// Channel to signal when tasks are done
	done := make(chan bool, 1)
	ctx := context.Background()
	// Run tasks in a goroutine
	go func() {
		defer func() {
			done <- true
		}()
		// works
		if len(os.Args[1:]) == 0 {
			showHelp()
			return
		}
		log.Println(os.Args[1])
		switch os.Args[1] {
		case "migration-files":
			feature_cli.GenerateMigrationFiles(ctx, false)
			feature_cli.GenerateMigrationFiles(ctx, true)
		case "restore-db":
			feature_cli.RestoreLocalDBFromMigrationFiles()
		default:
			showHelp()
		}
	}()
	// Wait for either task completion or interruption signal
	select {
	case <-done:
		fmt.Println()
	case <-quit:
		fmt.Println()
		log.Println("received interruption signal")
	}
	shutdown(logger)
}

func shutdown(loggerService *service.Logger) {
	// Shutdown telemetry service first to flush remaining telemetry
	log.Println("zeroize global keys")
	cfg.Default().Site.GlobalKeys.Zeroize()
	log.Println("shutting down telemetry service")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if telemetryService := loggerService.TelemetryService(); telemetryService != nil {
		if err := telemetryService.Shutdown(shutdownCtx); err != nil {
			loggerService.ErrorFunction(err, "telemetry service failed to shutdown")
		}
	}
	log.Println("server exiting")
}

func showHelp() {
	fmt.Println("Select one from the options:")
	fmt.Println("1. Generate migration files")
	fmt.Println("2. Restore local DB")
	var option string
	_, _ = fmt.Scan(&option)
	ctx := context.Background()
	switch option {
	case "1":
		feature_cli.GenerateMigrationFiles(ctx, false)
		feature_cli.GenerateMigrationFiles(ctx, true)
	case "2":
		feature_cli.RestoreLocalDBFromMigrationFiles()
	}
}
