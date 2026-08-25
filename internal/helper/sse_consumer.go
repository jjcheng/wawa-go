package helper

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// SSEEvent represents a single event received from an SSE stream
type SSEEvent struct {
	ID    string
	Event string
	Data  string
}

// SSEHandler is a function that processes an SSEEvent.
// Returns an error if the consumer should stop processing.
type SSEHandler func(event SSEEvent) error

// SSEConsumer handles server-to-server SSE communication
type SSEConsumer struct {
	URL         string
	Headers     map[string]string
	HTTPClient  *http.Client
	LastEventID string
	RetryDelay  time.Duration
	MaxRetries  int
	ReadTimeout time.Duration // Max time to wait for data before reconnecting
	Method      string        // HTTP method, defaults to GET
	Body        io.Reader     // Optional request body
}

// NewSSEConsumer creates a new SSE consumer with default settings
func NewSSEConsumer(url string) *SSEConsumer {
	return &SSEConsumer{
		URL:     url,
		Headers: make(map[string]string),
		HTTPClient: &http.Client{
			Timeout: 0, // SSE is long-lived
		},
		RetryDelay:  2 * time.Second,
		MaxRetries:  5,
		ReadTimeout: 30 * time.Second,
		Method:      "GET",
	}
}

// ErrStopConsume is a special error that can be returned by an SSEHandler
// to stop the consumption without it being treated as a failure.
var ErrStopConsume = fmt.Errorf("stop consumption")

// Consume starts the SSE connection and calls the handler for each event.
// It handles automatic reconnection using exponential backoff and Last-Event-ID.
func (c *SSEConsumer) Consume(ctx context.Context, handler SSEHandler) error {
	retries := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := c.connectAndRead(ctx, handler)
			if err != nil {
				if err == ErrStopConsume {
					return nil
				}
				if retries >= c.MaxRetries {
					return fmt.Errorf("max retries reached: %w", err)
				}
				retries++

				// Wait before retrying, respecting context
				timer := time.NewTimer(c.RetryDelay * time.Duration(retries))
				select {
				case <-ctx.Done():
					timer.Stop()
					return ctx.Err()
				case <-timer.C:
					continue
				}
			}

			// If connectAndRead returns nil, it means the connection closed (EOF).
			// SSE should automatically reconnect on EOF.
			// Re-zero retries as it was a successful (though now closed) connection.
			retries = 0
			continue
		}
	}
}

func (c *SSEConsumer) connectAndRead(ctx context.Context, handler SSEHandler) error {
	req, err := http.NewRequestWithContext(ctx, c.Method, c.URL, c.Body)
	if err != nil {
		return err
	}

	// Set required SSE headers
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Connection", "keep-alive")

	// Set user-defined headers
	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	// Resume from last seen ID if present
	if c.LastEventID != "" {
		req.Header.Set("Last-Event-ID", c.LastEventID)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	reader := bufio.NewReader(resp.Body)
	var currentEvent SSEEvent

	// Watchdog to handle ReadTimeout
	dataChan := make(chan struct{}, 1)
	if c.ReadTimeout > 0 {
		go func() {
			timer := time.NewTimer(c.ReadTimeout)
			defer timer.Stop()
			for {
				select {
				case <-dataChan:
					if !timer.Stop() {
						select {
						case <-timer.C:
						default:
						}
					}
					timer.Reset(c.ReadTimeout)
				case <-timer.C:
					resp.Body.Close() // Force close to break the reader
					return
				case <-ctx.Done():
					return
				}
			}
		}()
	}

	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			// Signal data received to reset watchdog
			select {
			case dataChan <- struct{}{}:
			default:
			}

			trimmedLine := strings.TrimSpace(line)
			if trimmedLine == "" {
				if currentEvent.Data != "" {
					// Save last event ID for reconnection
					if currentEvent.ID != "" {
						c.LastEventID = currentEvent.ID
					}

					if err := handler(currentEvent); err != nil {
						return err
					}
					currentEvent = SSEEvent{}
				}
			} else if strings.HasPrefix(trimmedLine, ":") {
				// ignore
			} else {
				parts := strings.SplitN(trimmedLine, ":", 2)
				key := parts[0]
				value := ""
				if len(parts) > 1 {
					value = strings.TrimLeft(parts[1], " ")
				}

				switch key {
				case "id":
					currentEvent.ID = value
				case "event":
					currentEvent.Event = value
				case "data":
					if currentEvent.Data != "" {
						currentEvent.Data += "\n"
					}
					currentEvent.Data += value
				case "retry":
					if dur, err := time.ParseDuration(value + "ms"); err == nil {
						c.RetryDelay = dur
					}
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				// If the stream ends and we have a partial event with data, process it
				if currentEvent.Data != "" {
					return handler(currentEvent)
				}
				return nil // Graceful close
			}
			return err
		}
	}
}
