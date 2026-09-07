package service

import (
	"sync"

	dto_wa "github.com/jjcheng/wawa-go/internal/dto/wa"
)

type WAMessageStream struct {
	mutex       sync.RWMutex
	subscribers map[string]map[chan WAMessageStreamEvent]struct{}
}

func NewWAMessageStream() *WAMessageStream {
	return &WAMessageStream{subscribers: make(map[string]map[chan WAMessageStreamEvent]struct{})}
}

type WAMessageStreamEvent struct {
	Type    string                     `json:"type"`
	Message *dto_wa.Message            `json:"message,omitempty"`
	Status  *dto_wa.MessageStatusEvent `json:"status,omitempty"`
}

func (stream *WAMessageStream) Subscribe(phoneNumberID string, customerPhoneNumber string) (<-chan WAMessageStreamEvent, func()) {
	channel := make(chan WAMessageStreamEvent, 32)
	key := waMessageStreamKey(phoneNumberID, customerPhoneNumber)
	stream.mutex.Lock()
	if stream.subscribers[key] == nil {
		stream.subscribers[key] = make(map[chan WAMessageStreamEvent]struct{})
	}
	stream.subscribers[key][channel] = struct{}{}
	stream.mutex.Unlock()
	return channel, func() {
		stream.mutex.Lock()
		delete(stream.subscribers[key], channel)
		if len(stream.subscribers[key]) == 0 {
			delete(stream.subscribers, key)
		}
		stream.mutex.Unlock()
	}
}

func (stream *WAMessageStream) PublishMessage(message dto_wa.Message) {
	key := waMessageStreamKey(message.PhoneNumberId, message.CustomerPhoneNumber)
	stream.publish(key, WAMessageStreamEvent{Type: "message", Message: &message})
}

func (stream *WAMessageStream) PublishStatus(message dto_wa.Message, status dto_wa.MessageStatusEvent) {
	key := waMessageStreamKey(message.PhoneNumberId, message.CustomerPhoneNumber)
	stream.publish(key, WAMessageStreamEvent{Type: "status", Status: &status})
}

func (stream *WAMessageStream) publish(key string, event WAMessageStreamEvent) {
	stream.mutex.RLock()
	defer stream.mutex.RUnlock()
	for channel := range stream.subscribers[key] {
		select {
		case channel <- event:
		default:
		}
	}
}

func waMessageStreamKey(phoneNumberID string, customerPhoneNumber string) string {
	return phoneNumberID + ":" + customerPhoneNumber
}
