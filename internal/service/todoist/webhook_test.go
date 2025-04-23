package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"example.com/bot/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestWebHookHandler_processWebHook(t *testing.T) {
	tests := []struct {
		name                string
		requestBody         models.WebHookRequest
		expectedOutput      models.WebHookParsed
		shouldSendToChannel bool
	}{
		{
			name: "Not completed item event",
			requestBody: models.WebHookRequest{
				EventName: "item:added",
				UserID:    "user123",
			},
			shouldSendToChannel: false,
		},
		{
			name: "Completed item with duration in minutes",
			requestBody: models.WebHookRequest{
				EventName: "item:completed",
				UserID:    "user123",
				EventData: models.Task{
					ID:      "task1",
					Content: "Test Task",
					Duration: &models.Duration{
						Amount: 30,
						Unit:   "minute",
					},
				},
			},
			expectedOutput: models.WebHookParsed{
				UserID:    "user123",
				TimeSpent: 30,
				AskTime:   false,
			},
			shouldSendToChannel: true,
		},
		{
			name: "Completed item with duration in days",
			requestBody: models.WebHookRequest{
				EventName: "item:completed",
				UserID:    "user123",
				EventData: models.Task{
					ID:      "task1",
					Content: "Test Task",
					Duration: &models.Duration{
						Amount: 1,
						Unit:   "day",
					},
				},
			},
			expectedOutput: models.WebHookParsed{
				UserID:    "user123",
				TimeSpent: 24 * 60, // 1 day = 24 hours * 60 minutes
				AskTime:   false,
			},
			shouldSendToChannel: true,
		},
		{
			name: "Completed item with @track label",
			requestBody: models.WebHookRequest{
				EventName: "item:completed",
				UserID:    "user123",
				EventData: models.Task{
					ID:      "task1",
					Content: "Test Task",
					Labels:  []string{"@track"},
				},
			},
			expectedOutput: models.WebHookParsed{
				UserID:    "user123",
				TimeSpent: 0,
				AskTime:   true,
			},
			shouldSendToChannel: true,
		},
		{
			name: "Completed item with @log label",
			requestBody: models.WebHookRequest{
				EventName: "item:completed",
				UserID:    "user123",
				EventData: models.Task{
					ID:      "task1",
					Content: "Test Task",
					Labels:  []string{"@log0205"}, // 2 hours and 5 minutes
				},
			},
			expectedOutput: models.WebHookParsed{
				UserID:    "user123",
				TimeSpent: (2 * 60) + 5, // 2 hours and 5 minutes in minutes
				AskTime:   false,
			},
			shouldSendToChannel: true,
		},
		{
			name: "Completed item without duration or relevant labels",
			requestBody: models.WebHookRequest{
				EventName: "item:completed",
				UserID:    "user123",
				EventData: models.Task{
					ID:      "task1",
					Content: "Test Task",
					Labels:  []string{"unrelated-label"},
				},
			},
			shouldSendToChannel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a channel to receive the output
			updates := make(chan models.WebHookParsed, 1)
			wh := NewWebHookHandler(updates)

			// Process the webhook
			wh.wg.Add(1)
			go wh.processWebHook(&tt.requestBody)

			if tt.shouldSendToChannel {
				// Wait for the output
				var output models.WebHookParsed
				select {
				case output = <-updates:
					assert.Equal(t, tt.expectedOutput.UserID, output.UserID)
					assert.Equal(t, tt.expectedOutput.TimeSpent, output.TimeSpent)
				case <-time.After(100 * time.Millisecond):
					t.Fatal("Timeout waiting for webhook processing")
				}
			} else {
				// Ensure nothing is sent to the channel
				select {
				case <-updates:
					t.Fatal("Unexpected output received from webhook processing")
				case <-time.After(100 * time.Millisecond):
					// This is expected, nothing should be sent to the channel
				}
			}
		})
	}
}

func TestWebHookHandler_handleHTTP(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "Valid request",
			requestBody: models.WebHookRequest{
				EventName: "item:completed",
				UserID:    "user123",
				EventData: models.Task{
					ID:      "task1",
					Content: "Test Task",
					Duration: &models.Duration{
						Amount: 30,
						Unit:   "minute",
					},
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid request body",
			requestBody:    "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a channel to receive the output
			updates := make(chan models.WebHookParsed, 1)
			wh := NewWebHookHandler(updates)

			// Create a test HTTP server
			handler := http.HandlerFunc(wh.handleHTTP)

			// Create a request
			var body []byte
			var err error

			if str, ok := tt.requestBody.(string); ok {
				body = []byte(str)
			} else {
				body, err = json.Marshal(tt.requestBody)
				assert.NoError(t, err)
			}

			req := httptest.NewRequest("POST", "/webhook", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			// Create a response recorder
			recorder := httptest.NewRecorder()

			// Serve the request
			handler.ServeHTTP(recorder, req)

			// Check the status code
			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}
