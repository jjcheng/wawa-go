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
	feature_broadcast "github.com/jjcheng/wawa-go/internal/feature/broadcast"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/setup"
)

var _dependenciesReadyChannel chan struct{}
var dependencies *service.Dependencies
var setupErr error

func main() {
	// set timezone to utc so no need to call .UTC() everytime
	time.Local = time.UTC
	// set log to stdout as error will be handled by loggerService
	log.SetOutput(os.Stdout)
	log.Printf("======== worker function invoked (%s) ========", cfg.Default().Site.Environment)
	// load dependencies
	_dependenciesReadyChannel = make(chan struct{})
	go func() {
		logger := service.NewLogger()
		defer close(_dependenciesReadyChannel)
		unitOfWork, err := setup.SetupDatabase(cfg.Default().Database.DSN(), logger)
		if err != nil {
			setupErr = fmt.Errorf("failed to setup database: %w", err)
			log.Print(setupErr)
			return
		}
		dependencies = service.NewDependencies(unitOfWork, logger)
	}()
	// trigger calls this api to invoke
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
		handleEvent(request.Context(), raw)
		// always return 200, we don't want FC to handle retries, use SMQ instead
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

// FC custom-runtime handler. SMQ triggers deliver one or more envelopes in an array.
func handleEvent(ctx context.Context, raw []byte) {
	// if set up dependencies have error, return error
	select {
	case <-_dependenciesReadyChannel:
		if setupErr != nil {
			return
		}
	case <-ctx.Done(): // if timeout or canceled
		return
	}
	log.Printf("worker received raw message: %s", raw)
	// sometimes body if an array, need to process one by one
	if body := bytes.TrimSpace(raw); len(body) > 0 && body[0] == '[' {
		var messages []json.RawMessage
		if err := json.Unmarshal(body, &messages); err != nil {
			dependencies.Logger.ErrorFunction(err)
			return
		}
		// process each message, error messages remain in SMQ for retry
		for _, message := range messages {
			processMessage(ctx, message)
		}
	} else {
		processMessage(ctx, raw)
	}
}

func processMessage(ctx context.Context, raw []byte) {
	message, err := handleMessage(ctx, raw)
	if err != nil {
		dependencies.Logger.ErrorFunction(err)
		return
	}
	// if no error when processing the message, delete the queue from SMQ; otherwise SMQ will re-send
	if err := deleteMessage(message); err != nil {
		dependencies.Logger.ErrorFunction(err)
	}
}

// Process one SMQ message envelope.
func handleMessage(ctx context.Context, raw []byte) (*service.MessageQueueMessage, error) {
	var message service.MessageQueueMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil, fmt.Errorf("handle_message failed to unmarshal raw to message: %w", err)
	}
	body := []byte(strings.TrimSpace(message.MessageBody))
	// in case it's base64 encoded
	if decodedBody, err := base64.StdEncoding.DecodeString(message.MessageBody); err == nil && json.Valid(decodedBody) {
		body = decodedBody
	}
	var queueJob service.MessageQueueJob
	if err := json.Unmarshal(body, &queueJob); err != nil {
		return &message, fmt.Errorf("handle_message failed to decode message body: %w", err)
	}
	// process job
	switch queueJob.Type {
	case "handle_wa_incoming":
		var rawBody string
		if err := json.Unmarshal(queueJob.Data, &rawBody); err != nil {
			return &message, fmt.Errorf("invalid handle_wa_incoming data: %w", err)
		}
		var incoming dto_wa.Incoming
		if err := json.Unmarshal([]byte(rawBody), &incoming); err != nil {
			return &message, fmt.Errorf("invalid handle_wa_incoming payload: %w", err)
		}
		err := feature_wa_message.ProcessIncoming(ctx, incoming, dependencies)
		if err != nil {
			return nil, fmt.Errorf("failed to handle_wa_incoming: %w", err)
		}
		return &message, nil
	case "retry_send_message":
		var messageId int32
		if err := json.Unmarshal(queueJob.Data, &messageId); err != nil {
			return &message, fmt.Errorf("invalid retry_send_message data %v: %w", queueJob.Data, err)
		}
		err := feature_wa_message.RetrySendingMessage(ctx, messageId, dependencies)
		if err != nil {
			return nil, fmt.Errorf("failed to retry_send_message: %w", err)
		}
		return &message, nil
	case "start_broadcast":
		var broadcastId int32
		if err := json.Unmarshal(queueJob.Data, &broadcastId); err != nil {
			return &message, fmt.Errorf("invalid start_broadcast data %v: %w", queueJob.Data, err)
		}
		err := feature_broadcast.Start(ctx, broadcastId, int(message.DequeueCount), dependencies)
		if err != nil {
			return nil, fmt.Errorf("failed to start_broadcast: %w", err)
		}
		return &message, nil
	}
	err := fmt.Errorf("unidentified handle_message queue job type: %s", queueJob.Type)
	return &message, err
}

func deleteMessage(message *service.MessageQueueMessage) error {
	if dependencies == nil || dependencies.MessageQueue == nil {
		log.Printf("worker message queue is not initialized")
		return nil
	}
	if message == nil {
		return nil
	}
	if strings.TrimSpace(message.ReceiptHandle) == "" {
		// return fmt.Errorf("queue message %s has no receipt handle", message.MessageID)
		//
		return nil
	}
	err := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle)
	if err != nil {
		return fmt.Errorf("failed to delete queue message %s: %w", message.ReceiptHandle, err)
	}
	return nil
}
