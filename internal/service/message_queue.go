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

type MessageQueuePriority int64

const (
	MessageQueuePriorityLowest  MessageQueuePriority = 1
	MessageQueuePriorityLow     MessageQueuePriority = 4
	MessageQueuePriorityNormal  MessageQueuePriority = 8
	MessageQueuePriorityHigh    MessageQueuePriority = 12
	MessageQueuePriorityHighest MessageQueuePriority = 16
)

type MessageQueueMessage struct {
	MessageID     string `json:"messageId"`
	ReceiptHandle string `json:"receiptHandle"`
	Body          string `json:"messageBody"`
	BodyMD5       string `json:"messageBodyMD5"`
	EnqueueTime   int64  `json:"enqueueTime"`
	DequeueCount  int64  `json:"dequeueCount"`
	Priority      int64  `json:"priority"`
}

type QueueJob struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
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

func (mq *MessageQueue) publishMessage(body string, delaySeconds int64, priority MessageQueuePriority) (string, error) {
	if strings.TrimSpace(body) == "" {
		return "", errors.New("message body cannot be empty")
	}
	if delaySeconds < 0 {
		delaySeconds = 0
	}
	if priority < MessageQueuePriorityLowest {
		priority = MessageQueuePriorityLowest
	}
	if priority > MessageQueuePriorityHighest {
		priority = MessageQueuePriorityHighest
	}
	encodedBody := base64.StdEncoding.EncodeToString([]byte(body))
	resp, err := mq.queue.SendMessage(ali_mns.MessageSendRequest{
		MessageBody:  encodedBody,
		DelaySeconds: delaySeconds,
		Priority:     int64(priority),
	})
	if err != nil {
		mq.logger.ErrorFunction(err, body)
		return "", fmt.Errorf("failed to publish message to queue %s: %w", cfg.Default().AliyunSMQ.QueueName, err)
	}
	mq.logger.Debugf("SMQ message published: queue=%s message_id=%s", cfg.Default().AliyunSMQ.QueueName, resp.MessageId)
	return resp.MessageId, nil
}

func (mq *MessageQueue) publishJSON(v any, delaySeconds int64, priority MessageQueuePriority) (string, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message payload: %w", err)
	}
	return mq.publishMessage(string(body), delaySeconds, priority)
}

func (mq *MessageQueue) PublishJob(jobType string, data any, delaySeconds int64, priority MessageQueuePriority) (string, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal queue job data: %w", err)
	}
	return mq.publishJSON(QueueJob{Type: jobType, Data: payload}, delaySeconds, priority)
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

func (mq *MessageQueue) ExtendMessageVisibility(message *MessageQueueMessage, visibilityTimeoutSeconds int64) error {
	if message == nil || strings.TrimSpace(message.ReceiptHandle) == "" {
		return errors.New("message receipt handle cannot be empty")
	}
	if visibilityTimeoutSeconds <= 0 {
		return errors.New("visibility timeout must be positive")
	}
	response, err := mq.queue.ChangeMessageVisibility(message.ReceiptHandle, visibilityTimeoutSeconds)
	if err != nil {
		return fmt.Errorf("failed to extend visibility for queue %s: %w", cfg.Default().AliyunSMQ.QueueName, err)
	}
	if strings.TrimSpace(response.ReceiptHandle) == "" {
		return errors.New("queue returned an empty receipt handle after visibility extension")
	}
	message.ReceiptHandle = response.ReceiptHandle
	return nil
}
