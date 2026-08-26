package service

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jjcheng/wawa-go/internal/cfg"

	ali_mns "github.com/aliyun/aliyun-mns-go-sdk"
)

type MessageQueue struct {
	logger *Logger
	client ali_mns.MNSClient
	queue  ali_mns.AliMNSQueue
}

type MessageQueueMessage struct {
	MessageID     string
	ReceiptHandle string
	Body          string
	BodyMD5       string
	EnqueueTime   int64
	DequeueCount  int64
	Priority      int64
}

func NewMessageQueue(logger *Logger) *MessageQueue {
	config := cfg.Default().AliyunSMQ
	queueName := strings.TrimSpace(config.QueueName)
	endpoint := strings.TrimSpace(config.Endpoint)
	if queueName == "" || endpoint == "" {
		panic("missing queue name or endpoint in message queue .env")
	}
	client := ali_mns.NewAliMNSClientWithConfig(ali_mns.AliMNSClientConfig{
		EndPoint:        cfg.Default().AliyunSMQ.Endpoint,
		AccessKeyId:     cfg.Default().AliyunSMQ.AccessKeyID,
		AccessKeySecret: cfg.Default().AliyunSMQ.AccessKeySecret,
	})
	queue := ali_mns.NewMNSQueue(queueName, client)
	return &MessageQueue{
		logger: logger,
		client: client,
		queue:  queue,
	}
}

func (mq *MessageQueue) PublishMessage(body string, delaySeconds int64, priority int64) (string, error) {
	if strings.TrimSpace(body) == "" {
		return "", errors.New("message body cannot be empty")
	}
	if delaySeconds < 0 {
		delaySeconds = 0
	}
	if priority < 0 {
		priority = 0
	}
	encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
	resp, err := mq.queue.SendMessage(ali_mns.MessageSendRequest{
		MessageBody:  encodedBody,
		DelaySeconds: delaySeconds,
		Priority:     priority,
	})
	if err != nil {
		return "", fmt.Errorf("failed to publish message to queue %s: %w", cfg.Default().AliyunSMQ.QueueName, err)
	}

	mq.logger.Debugf("SMQ message published: queue=%s message_id=%s", cfg.Default().AliyunSMQ.QueueName, resp.MessageId)
	return resp.MessageId, nil
}

func (mq *MessageQueue) PublishJSON(v any, delaySeconds int64, priority int64) (string, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message payload: %w", err)
	}
	return mq.PublishMessage(string(body), delaySeconds, priority)
}

func (mq *MessageQueue) ReceiveMessage(ctx context.Context) (*MessageQueueMessage, error) {
	respChan := make(chan ali_mns.MessageReceiveResponse, 1)
	errChan := make(chan error, 1)
	mq.queue.ReceiveMessage(respChan, errChan, cfg.Default().AliyunSMQ.PollingWaitSeconds)
	var timeout <-chan time.Time
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, context.DeadlineExceeded
		}
		timer := time.NewTimer(remaining)
		defer timer.Stop()
		timeout = timer.C
	}

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-timeout:
			return nil, context.DeadlineExceeded
		case err := <-errChan:
			if err == nil {
				continue
			}
			if ali_mns.ERR_MNS_MESSAGE_NOT_EXIST.IsEqual(err) {
				return nil, nil
			}
			errText := strings.ToLower(err.Error())
			if strings.Contains(errText, "messagenotexist") || strings.Contains(errText, "not exist") {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to receive message from queue %s: %w", cfg.Default().AliyunSMQ.QueueName, err)
		case resp := <-respChan:
			msg := &MessageQueueMessage{
				MessageID:     resp.MessageId,
				ReceiptHandle: resp.ReceiptHandle,
				Body:          resp.MessageBody,
				BodyMD5:       resp.MessageBodyMD5,
				EnqueueTime:   resp.EnqueueTime,
				DequeueCount:  resp.DequeueCount,
				Priority:      resp.Priority,
			}
			return msg, nil
		}
	}
}

func (mq *MessageQueue) DeleteMessage(receiptHandle string) error {
	if strings.TrimSpace(receiptHandle) == "" {
		return errors.New("receipt handle cannot be empty")
	}

	err := mq.queue.DeleteMessage(receiptHandle)
	if err != nil {
		return fmt.Errorf("failed to delete message from queue %s: %w", cfg.Default().AliyunSMQ.QueueName, err)
	}
	return nil
}
