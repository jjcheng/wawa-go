package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	"github.com/jjcheng/wawa-go/internal/controller"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/setup"
)

func main() {
	// set timezone to utc so no need to call .UTC() everytime
	time.Local = time.UTC
	// set log to stdout as error will be handled by loggerService
	log.SetOutput(os.Stdout)
	log.Printf("======== api function invoked (%s) ========", cfg.Default().Site.Environment)
	// setup database
	loggerService := service.NewLogger()
	unitOfWork, err := setup.SetupDatabase(cfg.Default().Database.DSN(), loggerService)
	if err != nil {
		log.Printf("FATAL: Failed to setup database: %v", err)
		panic(err.Error())
	}
	// setup services
	dependencies := service.NewDependencies(unitOfWork, loggerService)
	// setup router
	router := setup.SetupRouter(dependencies.Logger)
	// setup controllers
	controller.RegisterControllers(router, dependencies)
	// start server
	server := &http.Server{
		Addr:              ":" + cfg.Default().Site.Port,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       120 * time.Second,
	}
	go func() {
		log.Printf("server is running on port %s", cfg.Default().Site.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			loggerService.ErrorFunction(err, "server failed to start")
			panic(err.Error())
		}
	}()
	shutdown(loggerService, server)
}

func shutdown(loggerService *service.Logger, server *http.Server) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	log.Println("shutting down server")
	// shutdown the HTTP engine
	log.Println("shutting down HTTP engine")
	serverCtx, serverCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer serverCancel()
	if err := server.Shutdown(serverCtx); err != nil {
		loggerService.ErrorFunction(err, "HTTP server failed to shutdown gracefully")
		// force close if graceful shutdown fails
		if closeErr := server.Close(); closeErr != nil {
			loggerService.ErrorFunction(closeErr, "HTTP server failed to force close")
		}
		os.Exit(1)
	}
	log.Println("server exiting")
}
