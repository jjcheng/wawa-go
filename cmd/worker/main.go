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
	"github.com/jjcheng/wawa-go/internal/helper"
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
		dependencies = setup.SetupServices(unitOfWork, logger)
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
		return
	}
	// if no error when processing the message, delete the queue from SMQ; otherwise SMQ will re-send
	deleteMessage(message)
}

// Process one SMQ message envelope.
func handleMessage(ctx context.Context, raw []byte) (*service.MessageQueueMessage, error) {
	rawStr := string(raw)
	var message service.MessageQueueMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return nil, err
	}
	body := []byte(strings.TrimSpace(message.MessageBody))
	// in case it's base64 encoded
	if decodedBody, err := base64.StdEncoding.DecodeString(message.MessageBody); err == nil && json.Valid(decodedBody) {
		body = decodedBody
	}
	queueJob, err := helper.DeserializeJSON[service.MessageQueueJob](string(body))
	if err != nil {
		logWorkerError(err, message.MessageID)
		return &message, err
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
		return &message, feature_wa_message.ProcessIncoming(ctx, incoming, dependencies)
	case "retry_send_message":
		var messageId int32
		if err := json.Unmarshal(queueJob.Data, &messageId); err != nil {
			return &message, fmt.Errorf("invalid retry_send_message data: %w", err)
		}
		err := feature_wa_message.RetrySendingMessage(ctx, messageId, dependencies)
		return &message, err
	case "start_broadcast":
		var broadcastId int32
		if err := json.Unmarshal(queueJob.Data, &broadcastId); err != nil {
			return &message, fmt.Errorf("invalid start_broadcast data: %w", err)
		}
		err := feature_broadcast.Process(ctx, broadcastId, int(message.DequeueCount), dependencies)
		return &message, err
	}
	err = fmt.Errorf("unidentified type: %s", rawStr)
	dependencies.Logger.ErrorFunction(err, rawStr)
	return &message, err
}

func deleteMessage(message *service.MessageQueueMessage) {
	if dependencies == nil || dependencies.MessageQueue == nil {
		log.Printf("worker message queue is not initialized")
		return
	}
	if message == nil {
		return
	}
	if strings.TrimSpace(message.ReceiptHandle) == "" {
		dependencies.Logger.ErrorFunction(fmt.Errorf("SMQ message %s has no receipt handle", message.MessageID))
		return
	}
	err := dependencies.MessageQueue.DeleteMessage(message.ReceiptHandle)
	if err != nil {
		dependencies.Logger.ErrorFunction(err, message.ReceiptHandle)
	}
}

func logWorkerError(err error, args ...any) {
	if dependencies != nil && dependencies.Logger != nil {
		dependencies.Logger.ErrorFunction(err, args...)
		return
	}
	log.Printf("worker error: %v args=%v", err, args)
}
