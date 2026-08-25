package helper

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSSEConsumer_Consume(t *testing.T) {
	// 1. Mock SSE Server
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")
		w.WriteHeader(http.StatusOK)

		flusher, _ := w.(http.Flusher)

		fmt.Fprint(w, "id: 1\n")
		fmt.Fprint(w, "event: message\n")
		fmt.Fprint(w, "data: hello world\n\n")
		if flusher != nil {
			flusher.Flush()
		}

		fmt.Fprint(w, "id: 2\n")
		fmt.Fprint(w, "data: multiline\n")
		fmt.Fprint(w, "data: data\n\n")
		if flusher != nil {
			flusher.Flush()
		}

		// Small delay to ensure client can read before connection closes
		time.Sleep(50 * time.Millisecond)
	}))
	defer ts.Close()

	// 2. Setup Consumer
	consumer := NewSSEConsumer(ts.URL)
	var events []SSEEvent

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := consumer.Consume(ctx, func(event SSEEvent) error {
		events = append(events, event)
		if len(events) >= 2 {
			return ErrStopConsume // Gracefully stop
		}
		return nil
	})

	// 3. Assertions
	assert.NoError(t, err)
	assert.Len(t, events, 2)

	assert.Equal(t, "1", events[0].ID)
	assert.Equal(t, "message", events[0].Event)
	assert.Equal(t, "hello world", events[0].Data)

	assert.Equal(t, "2", events[1].ID)
	assert.Equal(t, "multiline\ndata", events[1].Data)
	assert.Equal(t, "2", consumer.LastEventID)
}

func TestSSEConsumer_Reconnect(t *testing.T) {
	connectionCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connectionCount++
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		if connectionCount == 1 {
			// First connection sends one event then closes (graceful EOF)
			fmt.Fprint(w, "id: 100\n")
			fmt.Fprint(w, "data: first\n\n")
			return
		}

		// Second connection verifies Last-Event-ID
		lastID := r.Header.Get("Last-Event-ID")
		if lastID == "100" {
			fmt.Fprint(w, "id: 101\n")
			fmt.Fprint(w, "data: success\n\n")
		}
	}))
	defer ts.Close()

	consumer := NewSSEConsumer(ts.URL)
	consumer.RetryDelay = 10 * time.Millisecond
	consumer.MaxRetries = 2

	var events []SSEEvent
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	consumer.Consume(ctx, func(event SSEEvent) error {
		events = append(events, event)
		if event.Data == "success" {
			return ErrStopConsume
		}
		return nil
	})

	assert.Len(t, events, 2)
	assert.Equal(t, "first", events[0].Data)
	assert.Equal(t, "success", events[1].Data)
	assert.GreaterOrEqual(t, connectionCount, 2)
}
