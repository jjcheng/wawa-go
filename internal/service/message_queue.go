package service

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

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
	MessageID      string `json:"messageId"`
	ReceiptHandle  string `json:"receiptHandle"`
	MessageBody    string `json:"messageBody"`
	MessageBodyMD5 string `json:"messageBodyMD5"`
	EnqueueTime    int64  `json:"enqueueTime"`
	DequeueCount   int64  `json:"dequeueCount"`
	Priority       int64  `json:"priority"`
}

type MessageQueueJob struct {
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
		return "", fmt.Errorf("MessageQueue.publishMessage body=%s delaySeconds=%d priority=%v error=%w", body, delaySeconds, priority, err)
	}
	mq.logger.Debugf("SMQ message published: queue=%s message_id=%s", cfg.Default().AliyunSMQ.QueueName, resp.MessageId)
	return resp.MessageId, nil
}

func (mq *MessageQueue) publishJSON(v any, delaySeconds int64, priority MessageQueuePriority) (string, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return "", fmt.Errorf("MessageQueue.publishJSON v=%v delaySeconds=%d priority=%v error=%w", v, delaySeconds, priority, err)
	}
	return mq.publishMessage(string(body), delaySeconds, priority)
}

func (mq *MessageQueue) PublishJob(jobType string, data any, delaySeconds int64, priority MessageQueuePriority) (string, error) {
	payload, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("MessageQueue.PublishJob jobType=%s data=%v delaySeconds=%d priority=%v error=%w", jobType, data, delaySeconds, priority, err)
	}
	messageId, err := mq.publishJSON(MessageQueueJob{Type: jobType, Data: payload}, delaySeconds, priority)
	if err != nil {
		return "", fmt.Errorf("MessageQueue.PublishJob jobType=%s data=%v delaySeconds=%d priority=%v error=%w", jobType, data, delaySeconds, priority, err)
	}
	return messageId, nil
}

func (mq *MessageQueue) DeleteMessage(receiptHandle string) error {
	if strings.TrimSpace(receiptHandle) == "" {
		return fmt.Errorf("MessageQueue.DeleteMessage index=0 recipientHandle=%s error=%w", receiptHandle, errors.New("recipient handle is empty"))
	}
	err := mq.queue.DeleteMessage(receiptHandle)
	if err != nil {
		return fmt.Errorf("Message`Queue.DeleteMessage index=1 recipientHandle=%s error=%w", receiptHandle, err)
	}
	return nil
}

func (mq *MessageQueue) ExtendMessageVisibility(message *MessageQueueMessage, visibilityTimeoutSeconds int64) error {
	if message == nil || strings.TrimSpace(message.ReceiptHandle) == "" {
		return fmt.Errorf("MessageQueue.ExtendMessageVisibility index=0 messageId=%s visibilityTimeoutSeconds=%d error=%w", message.MessageID, visibilityTimeoutSeconds, errors.New("message is nil or message.ReciptHandle is empty"))
	}
	if visibilityTimeoutSeconds <= 0 {
		return errors.New("visibility timeout must be positive")
	}
	response, err := mq.queue.ChangeMessageVisibility(message.ReceiptHandle, visibilityTimeoutSeconds)
	if err != nil {
		return fmt.Errorf("MessageQueue.ExtendMessageVisibility index=1 messageId=%s visibilityTimeoutSeconds=%d error=%w", message.MessageID, visibilityTimeoutSeconds, err)
	}
	if strings.TrimSpace(response.ReceiptHandle) == "" {
		return errors.New("queue returned an empty receipt handle after visibility extension")
	}
	message.ReceiptHandle = response.ReceiptHandle
	return nil
}
