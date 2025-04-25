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
		requestBody         *models.WebHookRequest
		expectedOutput      models.WebHookParsed
		shouldSendToChannel bool
	}{
		{
			name: "Not completed item event",
			requestBody: createWebhookRequest("item:added", "user123", models.Task{
				ID:      "task1",
				Content: "Test Task",
			}),
			shouldSendToChannel: false,
		},
		{
			name: "Completed item with duration in minutes",
			requestBody: createWebhookRequest("item:completed", "user123", models.Task{
				ID:      "task1",
				Content: "Test Task",
				Duration: &models.Duration{
					Amount: 30,
					Unit:   "minute",
				},
			}),
			expectedOutput: models.WebHookParsed{
				UserID:    "user123",
				Task:      "Test Task",
				TimeSpent: 30,
				AskTime:   false,
			},
			shouldSendToChannel: true,
		},
		{
			name: "Completed item with duration in days",
			requestBody: createWebhookRequest("item:completed", "user123", models.Task{
				ID:      "task1",
				Content: "Test Task",
				Duration: &models.Duration{
					Amount: 1,
					Unit:   "day",
				},
			}),
			expectedOutput: models.WebHookParsed{
				UserID:    "user123",
				Task:      "Test Task",
				TimeSpent: 24 * 60,
				AskTime:   false,
			},
			shouldSendToChannel: true,
		},
		{
			name: "Completed item with @track label",
			requestBody: createWebhookRequest("item:completed", "user123", models.Task{
				ID:      "task1",
				Content: "Test Task",
				Labels:  []string{"track"},
			}),
			expectedOutput: models.WebHookParsed{
				UserID:    "user123",
				Task:      "Test Task",
				TimeSpent: 0,
				AskTime:   true,
			},
			shouldSendToChannel: true,
		},
		{
			name: "Completed item with @log label",
			requestBody: createWebhookRequest("item:completed", "user123", models.Task{
				ID:      "task1",
				Content: "Test Task",
				Labels:  []string{"log0205"},
			}),
			expectedOutput: models.WebHookParsed{
				UserID:    "user123",
				Task:      "Test Task",
				TimeSpent: (2 * 60) + 5,
				AskTime:   false,
			},
			shouldSendToChannel: true,
		},
		{
			name: "Completed item without duration or relevant labels",
			requestBody: createWebhookRequest("item:completed", "user123", models.Task{
				ID:      "task1",
				Content: "Test Task",
				Labels:  []string{"unrelated-label"},
			}),
			shouldSendToChannel: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updates := make(chan models.WebHookParsed, 1)
			wh := NewWebHookHandler(updates)

			wh.wg.Add(1)
			go wh.processWebHook(tt.requestBody)

			if tt.shouldSendToChannel {
				var output models.WebHookParsed
				select {
				case output = <-updates:
					assert.Equal(t, tt.expectedOutput.UserID, output.UserID)
					assert.Equal(t, tt.expectedOutput.Task, output.Task)
					assert.Equal(t, tt.expectedOutput.TimeSpent, output.TimeSpent)
					assert.Equal(t, tt.expectedOutput.AskTime, output.AskTime)
				case <-time.After(100 * time.Millisecond):
					t.Fatal("Timeout waiting for webhook processing")
				}
			} else {
				select {
				case <-updates:
					t.Fatal("Unexpected output received from webhook processing")
				case <-time.After(100 * time.Millisecond):
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
			requestBody: createWebhookRequestRaw("item:completed", "user123", models.Task{
				ID:      "task1",
				Content: "Test Task",
				Duration: &models.Duration{
					Amount: 30,
					Unit:   "minute",
				},
			}),
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
			updates := make(chan models.WebHookParsed, 1)
			wh := NewWebHookHandler(updates)

			handler := http.HandlerFunc(wh.handleHTTP)

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

			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, req)

			assert.Equal(t, tt.expectedStatus, recorder.Code)
		})
	}
}

func createWebhookRequest(eventName string, userID string, task models.Task) *models.WebHookRequest {
	taskData, _ := json.Marshal(task)
	return &models.WebHookRequest{
		EventName: eventName,
		UserID:    userID,
		EventData: taskData,
	}
}

func createWebhookRequestRaw(eventName string, userID string, task models.Task) models.WebHookRequest {
	taskData, _ := json.Marshal(task)
	return models.WebHookRequest{
		EventName: eventName,
		UserID:    userID,
		EventData: taskData,
	}
}
