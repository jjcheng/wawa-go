package main

import (
	"bytes"
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
	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
	feature_campaign "github.com/jjcheng/wawa-go/internal/feature/campaign"
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
	log.Printf("======== worker function invoked (%s) ========", cfg.Default().Site.Environment)

	dependenciesReadyChannel := make(chan struct{})
	var dependencies *service.Dependencies
	var setupErr error
	go func() {
		logger := service.NewLogger()
		defer close(dependenciesReadyChannel)
		unitOfWork, err := setup.SetupDatabase(cfg.Default().Database.DSN(), logger)
		if err != nil {
			setupErr = fmt.Errorf("failed to setup database: %w", err)
			log.Print(setupErr)
			return
		}
		dependencies = setup.SetupServices(unitOfWork, logger)
	}()

	// Process one SMQ message envelope.
	handleMessage := func(ctx context.Context, raw []byte) error {
		rawStr := string(raw)
		var message service.MessageQueueMessage
		if err := json.Unmarshal(raw, &message); err != nil {
			return err
		}
		body := []byte(strings.TrimSpace(message.MessageBody))
		// in case it's base64 encoded
		if decodedBody, err := base64.StdEncoding.DecodeString(message.MessageBody); err == nil && json.Valid(decodedBody) {
			body = decodedBody
		}
		queueJob, err := helper.DeserializeJSON[service.MessageQueueJob](string(body))
		if err != nil {
			dependencies.Logger.ErrorFunction(err, message.MessageID)
			return err
		}
		switch queueJob.Type {
		case "handle_wa_incoming":
			var rawBody string
			if err := json.Unmarshal(queueJob.Data, &rawBody); err != nil {
				return fmt.Errorf("invalid handle_wa_incoming data: %w", err)
			}
			var incoming dto_wa.Incoming
			if err := json.Unmarshal([]byte(rawBody), &incoming); err != nil {
				return fmt.Errorf("invalid handle_wa_incoming payload: %w", err)
			}
			return feature_wa_message.ProcessIncoming(ctx, incoming, &message, dependencies)
		case "retry_send_message":
			var messageId int32
			if err := json.Unmarshal(queueJob.Data, &messageId); err != nil {
				return fmt.Errorf("invalid retry_send_message data: %w", err)
			}
			return feature_wa_message.RetrySendingMessage(ctx, messageId, &message, dependencies)
		case "start_campaign":
			var campaignID int32
			if err := json.Unmarshal(queueJob.Data, &campaignID); err != nil {
				return fmt.Errorf("invalid start_campaign data: %w", err)
			}
			return feature_campaign.Process(ctx, campaignID, dependencies)
		}
		return fmt.Errorf("unidentified type: %s", rawStr)
	}

	// FC custom-runtime handler. SMQ triggers deliver one or more envelopes in an array.
	handleEvent := func(ctx context.Context, raw []byte) error {
		select {
		case <-dependenciesReadyChannel:
			if setupErr != nil {
				return setupErr
			}
		case <-ctx.Done():
			return ctx.Err()
		}
		if body := bytes.TrimSpace(raw); len(body) > 0 && body[0] == '[' {
			var messages []json.RawMessage
			if err := json.Unmarshal(body, &messages); err != nil {
				return fmt.Errorf("invalid SMQ message batch: %w", err)
			}
			// handle multiple messages
			for _, message := range messages {
				if err := handleMessage(ctx, message); err != nil {
					return err
				}
			}
			return nil
		}
		// handle message
		return handleMessage(ctx, raw)
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
			dependencies.Logger.ErrorFunction(err, request.Header.Get("X-Fc-Request-Id"), string(raw))
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
		response.WriteHeader(http.StatusOK)
	})

	// health check
	http.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusOK)
	})

	port := cfg.Default().Site.Port
	log.Printf("starting worker function HTTP server on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
