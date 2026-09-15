package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/aliyun/fc-runtime-go-sdk/fc"
	"github.com/jjcheng/wawa-go/internal/cfg"
	feature_campaign "github.com/jjcheng/wawa-go/internal/feature/campaign"
	"github.com/jjcheng/wawa-go/internal/service"
	"github.com/jjcheng/wawa-go/internal/setup"
)

type EventBridgeEvent struct {
	ID              string         `json:"id"`
	Source          string         `json:"source"`
	Type            string         `json:"type"`
	Subject         string         `json:"subject"`
	Time            string         `json:"time"`
	EventSourceName string         `json:"eventsourcename"`
	Task            string         `json:"task"` // campaign
	Data            map[string]any `json:"data"`
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

	// If CLI arguments are provided, run the task directly (useful for local CLI testing)
	if len(os.Args) > 1 && os.Args[1] != "" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		runCLITask(ctx, os.Args[1:], dependencies)
		return
	}

	// Standard Function Compute Go Runtime Handler
	handleEvent := func(ctx context.Context, event EventBridgeEvent) (string, error) {
		campaignId := parseCampaignIdFromEvent(event)
		if campaignId <= 0 {
			err := fmt.Errorf("no valid campaign id found in event: %+v", event)
			dependencies.Logger.Error(err)
			return "", err
		}
		dependencies.Logger.Infof("FC received event: type=%s, subject=%s, campaignId=%d\n", event.Type, event.Subject, campaignId)
		if err := feature_campaign.Process(ctx, campaignId, dependencies); err != nil {
			return "", err
		}
		return fmt.Sprintf("campaign %d processed successfully", campaignId), nil
	}
	fc.Start(handleEvent)
}

func parseCampaignIdFromEvent(event EventBridgeEvent) int32 {
	// 1. Check direct "id" or "campaign_id" in event.Data
	if len(event.Data) > 0 {
		if id, ok := event.Data["id"]; ok {
			if parsed := convertToInt32(id); parsed > 0 {
				return parsed
			}
		}
		if id, ok := event.Data["campaign_id"]; ok {
			if parsed := convertToInt32(id); parsed > 0 {
				return parsed
			}
		}
	}
	// 2. Check event.EventSourceName, event.Source, or event.Subject for prefix "campaign-"
	for _, text := range []string{event.EventSourceName, event.Source, event.Subject} {
		if strings.HasPrefix(text, "campaign-") {
			if id, err := strconv.Atoi(strings.TrimPrefix(text, "campaign-")); err == nil && id > 0 {
				return int32(id)
			}
		}
	}
	return 0
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

func runCLITask(ctx context.Context, args []string, dependencies *service.Dependencies) {
	log.Println("running CLI worker task:", args[0])
	switch args[0] {
	case "campaign":
		if len(args) < 2 || args[1] == "" {
			log.Println("missing campaign ID argument")
			return
		}
		id, err := strconv.Atoi(args[1])
		if err != nil || id <= 0 {
			dependencies.Logger.Warnf("invalid campaign ID: %s\n", args[1])
			return
		}
		if err := feature_campaign.Process(ctx, int32(id), dependencies); err != nil {
			dependencies.Logger.ErrorFunction(err, id)
		}
	default:
		dependencies.Logger.Warnf("unknown worker task: %s\n", args[0])
	}
}
