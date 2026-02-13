package notifications

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestNewWhatsAppCloudSender(t *testing.T) {
	tests := []struct {
		name          string
		accessToken   string
		phoneNumberID string
		wantErr       bool
	}{
		{
			name:          "Valid credentials",
			accessToken:   "test_token",
			phoneNumberID: "123456789",
			wantErr:       false,
		},
		{
			name:          "Missing access token",
			accessToken:   "",
			phoneNumberID: "123456789",
			wantErr:       true,
		},
		{
			name:          "Missing phone number ID",
			accessToken:   "test_token",
			phoneNumberID: "",
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("WHATSAPP_ACCESS_TOKEN", tt.accessToken)
			os.Setenv("WHATSAPP_PHONE_NUMBER_ID", tt.phoneNumberID)
			defer os.Unsetenv("WHATSAPP_ACCESS_TOKEN")
			defer os.Unsetenv("WHATSAPP_PHONE_NUMBER_ID")

			sender, err := NewWhatsAppCloudSender()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewWhatsAppCloudSender() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && sender == nil {
				t.Error("NewWhatsAppCloudSender() returned nil sender")
			}
		})
	}
}

func TestWhatsAppCloudSender_SendTemplate(t *testing.T) {
	tests := []struct {
		name           string
		to             string
		templateName   string
		languageCode   string
		parameters     []string
		mockStatusCode int
		mockResponse   WhatsAppResponse
		wantErr        bool
	}{
		{
			name:           "Successful template send",
			to:             "+2348001234567",
			templateName:   "appointment_confirmation",
			languageCode:   "en_US",
			parameters:     []string{"Monday, Feb 10", "2:00 PM", "Lagos General"},
			mockStatusCode: http.StatusOK,
			mockResponse: WhatsAppResponse{
				MessagingProduct: "whatsapp",
				Messages: []struct {
					ID string `json:"id"`
				}{
					{ID: "wamid.test123"},
				},
			},
			wantErr: false,
		},
		{
			name:           "API error response",
			to:             "+2348001234567",
			templateName:   "appointment_confirmation",
			languageCode:   "en_US",
			parameters:     []string{"Monday, Feb 10"},
			mockStatusCode: http.StatusBadRequest,
			mockResponse:   WhatsAppResponse{},
			wantErr:        true,
		},
		{
			name:           "Empty parameters",
			to:             "+2348001234567",
			templateName:   "appointment_confirmation",
			languageCode:   "en_US",
			parameters:     []string{},
			mockStatusCode: http.StatusOK,
			mockResponse: WhatsAppResponse{
				Messages: []struct {
					ID string `json:"id"`
				}{
					{ID: "wamid.test456"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Verify request method and headers
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
				}

				w.WriteHeader(tt.mockStatusCode)
				if err := json.NewEncoder(w).Encode(tt.mockResponse); err != nil {
					t.Errorf("failed to encode mock response: %v", err)
				}
			}))
			defer server.Close()

			// Create sender with mock server - override the base URL
			sender := &WhatsAppCloudSender{
				accessToken:   "test_token",
				phoneNumberID: "123456789",
				httpClient:    server.Client(),
				baseURL:       server.URL, // Use mock server URL
			}

			messageID, err := sender.SendTemplate(tt.to, tt.templateName, tt.languageCode, tt.parameters)

			if (err != nil) != tt.wantErr {
				t.Errorf("SendTemplate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && messageID == "" {
				t.Error("SendTemplate() returned empty message ID")
			}
		})
	}
}

func TestWhatsAppCloudSender_SendText(t *testing.T) {
	tests := []struct {
		name           string
		to             string
		body           string
		mockStatusCode int
		mockResponse   WhatsAppResponse
		wantErr        bool
	}{
		{
			name:           "Successful text send",
			to:             "+2348001234567",
			body:           "Your appointment is confirmed",
			mockStatusCode: http.StatusOK,
			mockResponse: WhatsAppResponse{
				Messages: []struct {
					ID string `json:"id"`
				}{
					{ID: "wamid.text123"},
				},
			},
			wantErr: false,
		},
		{
			name:           "Empty body",
			to:             "+2348001234567",
			body:           "",
			mockStatusCode: http.StatusOK,
			mockResponse: WhatsAppResponse{
				Messages: []struct {
					ID string `json:"id"`
				}{
					{ID: "wamid.text456"},
				},
			},
			wantErr: false,
		},
		{
			name:           "API rate limit error",
			to:             "+2348001234567",
			body:           "Test message",
			mockStatusCode: http.StatusTooManyRequests,
			mockResponse:   WhatsAppResponse{},
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.mockStatusCode)
				if err := json.NewEncoder(w).Encode(tt.mockResponse); err != nil {
					t.Errorf("failed to encode mock response: %v", err)
				}
			}))
			defer server.Close()

			sender := &WhatsAppCloudSender{
				accessToken:   "test_token",
				phoneNumberID: "123456789",
				httpClient:    server.Client(),
				baseURL:       server.URL, // Use mock server URL
			}

			messageID, err := sender.SendText(tt.to, tt.body)

			if (err != nil) != tt.wantErr {
				t.Errorf("SendText() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && messageID == "" {
				t.Error("SendText() returned empty message ID")
			}
		})
	}
}

func TestWhatsAppCloudSender_SendMessage_NetworkError(t *testing.T) {
	sender := &WhatsAppCloudSender{
		accessToken:   "test_token",
		phoneNumberID: "123456789",
		httpClient:    &http.Client{},
	}

	// Use invalid URL to simulate network error
	message := WhatsAppTextMessage{
		MessagingProduct: "whatsapp",
		To:               "+2348001234567",
	}

	_, err := sender.sendMessage(message)
	if err == nil {
		t.Error("Expected network error, got nil")
	}
}

func TestWhatsAppResponse_NoMessageID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(WhatsAppResponse{
			Messages: []struct {
				ID string `json:"id"`
			}{}, // Empty messages array
		}); err != nil {
			t.Errorf("failed to encode mock response: %v", err)
		}
	}))
	defer server.Close()

	sender := &WhatsAppCloudSender{
		accessToken:   "test_token",
		phoneNumberID: "123456789",
		baseURL:       server.URL, // Use mock server URL
		httpClient:    server.Client(),
	}

	_, err := sender.SendText("+2348001234567", "Test")
	if err == nil {
		t.Error("Expected error for missing message ID, got nil")
	}
}
