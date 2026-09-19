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
	feature_cli "github.com/jjcheng/wawa-go/internal/feature/cli"
)

func main() {
	// set timezone to utc so no need to call .UTC() everytime
	time.Local = time.UTC
	// set log to stdout as error will be handled by loggerService
	log.SetOutput(os.Stdout)
	log.Println("starting cli")
	log.Printf("environment: %s\n", cfg.Default().Site.Environment)
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
			feature_cli.GenerateMigrationFiles(ctx)
		case "restore-db":
			feature_cli.RestoreLocalDBFromMigrationFiles()
		case "backfill-encryption":
			if err := feature_cli.BackfillEncryption(ctx); err != nil {
				log.Fatalf("encryption backfill failed: %v", err)
			}
		case "backfill-messages":
			if err := feature_cli.BackfillMessages(ctx); err != nil {
				log.Fatalf("message encryption backfill failed: %v", err)
			}
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
		feature_cli.GenerateMigrationFiles(ctx)
	case "2":
		feature_cli.RestoreLocalDBFromMigrationFiles()
	}
}
