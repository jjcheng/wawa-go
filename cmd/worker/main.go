package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aliyun/fc-runtime-go-sdk/fc"
	"github.com/jjcheng/wawa-go/internal/cfg"
	feature_wa_message "github.com/jjcheng/wawa-go/internal/feature/wa/message"
	"github.com/jjcheng/wawa-go/internal/helper"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/setup"
)

type EventBridgeEvent struct {
	ID              string            `json:"id"`
	Source          string            `json:"source"`
	Type            string            `json:"type"`
	Subject         string            `json:"subject"`
	Time            string            `json:"time"`
	EventSourceName string            `json:"eventsourcename"`
	Data            map[string]string `json:"data"` // task: "campaign"
}

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

	// Standard Function Compute Go Runtime Handler
	handleEvent := func(ctx context.Context, raw []byte) error {
		rawStr := string(raw)
		dependencies.Logger.Infof("new task: %s", rawStr)
		var envelope map[string]any
		if err := json.Unmarshal(raw, &envelope); err != nil {
			return err
		}
		// triggered by time
		if _, ok := envelope["triggerName"].(string); ok {

		} else if _, ok := envelope["messageBody"]; ok { // triggered by SMQ
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
	fc.Start(handleEvent)
}

func parseCampaignIdFromEvent(event EventBridgeEvent) (int32, error) {
	if len(event.Data) > 0 {
		if id, ok := event.Data["id"]; ok {
			if parsed := convertToInt32(id); parsed > 0 {
				return parsed, nil
			}
		}
	}
	return 0, fmt.Errorf("unable to parse campaign id: %v", event.Data)
}

func convertToInt32(v any) int32 {
	switch n := v.(type) {
	case float64:
		return int32(n)
	case int:
		return int32(n)
	case int32:
		return n
	case int64:
		return int32(n)
	case string:
		if id, err := strconv.Atoi(n); err == nil {
			return int32(id)
		}
	}
	return 0
}
