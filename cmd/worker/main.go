package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	"github.com/jjcheng/wawa-go/internal/helper"
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

	logger := service.NewLogger()
	dependenciesReady := make(chan struct{})
	var dependencies *service.Dependencies
	var setupErr error
	go func() {
		defer close(dependenciesReady)
		unitOfWork, err := setup.SetupDatabase(cfg.Default().Database.DSN(), logger)
		if err != nil {
			setupErr = fmt.Errorf("failed to setup database: %w", err)
			log.Print(setupErr)
			return
		}
		dependencies = setup.SetupServices(unitOfWork, logger)
	}()

	// FC custom runtime handler. dispatcher function will be triggered by time-trigger here
	handleEvent := func(ctx context.Context, raw []byte) error {
		select {
		case <-dependenciesReady:
			if setupErr != nil {
				return setupErr
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		rawStr := string(raw)
		logger.Infof("======== NEW TASK ========\n%s", rawStr)
		var envelope map[string]any
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return err
		}
		if _, ok := envelope["messageBody"]; ok { // triggered by SMQ
			var message service.MessageQueueMessage
			if err := json.Unmarshal(raw, &message); err != nil {
				return err
			}
			body := []byte(strings.TrimSpace(message.Body))
			if decodedBody, err := base64.StdEncoding.DecodeString(message.Body); err == nil && json.Valid(decodedBody) {
				body = decodedBody
			}
			queueJob, err := helper.DeserializeJSON[service.QueueJob](string(body))
			if err != nil {
				dependencies.Logger.ErrorFunction(err, message.MessageID)
				return err
			}
			switch queueJob.Type {
			case "wa_incoming":
				return feature_wa_message.ProcessIncoming(ctx, string(queueJob.Data), &message, dependencies)
			case "retry_send_message":
				return feature_wa_message.RetrySendingMessage(ctx, string(queueJob.Data), &message, dependencies)
			}
		}
		return fmt.Errorf("unidentified type: %s", rawStr)
	}

	http.HandleFunc("/invoke", func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			response.Header().Set("Allow", http.MethodPost)
			http.Error(response, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		raw, err := io.ReadAll(request.Body)
		if err != nil {
			http.Error(response, "failed to read request body", http.StatusBadRequest)
			return
		}
		if err := handleEvent(request.Context(), raw); err != nil {
			dependencies.Logger.ErrorFunction(err, request.Header.Get("X-Fc-Request-Id"))
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
		response.WriteHeader(http.StatusOK)
	})

	http.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	})

	port := cfg.Default().Site.Port
	log.Printf("starting worker function HTTP server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
