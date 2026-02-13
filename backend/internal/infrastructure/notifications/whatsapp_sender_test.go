package notifications

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func newMockClient(t *testing.T, statusCode int, response WhatsAppResponse, validate func(*http.Request, []byte)) *http.Client {
	t.Helper()

	return &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			requestBody, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			if validate != nil {
				validate(req, requestBody)
			}

			respBody, err := json.Marshal(response)
			if err != nil {
				return nil, err
			}

			return &http.Response{
				StatusCode: statusCode,
				Header:     make(http.Header),
				Body:       io.NopCloser(bytes.NewReader(respBody)),
			}, nil
		}),
	}
}

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
			client := newMockClient(t, tt.mockStatusCode, tt.mockResponse, func(r *http.Request, _ []byte) {
				// Verify request method and headers
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				if r.Header.Get("Content-Type") != "application/json" {
					t.Errorf("Expected Content-Type: application/json, got %s", r.Header.Get("Content-Type"))
				}
			})

			sender := &WhatsAppCloudSender{
				accessToken:   "test_token",
				phoneNumberID: "123456789",
				httpClient:    client,
				baseURL:       "https://mock.whatsapp.local",
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
			client := newMockClient(t, tt.mockStatusCode, tt.mockResponse, nil)

			sender := &WhatsAppCloudSender{
				accessToken:   "test_token",
				phoneNumberID: "123456789",
				httpClient:    client,
				baseURL:       "https://mock.whatsapp.local",
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
		httpClient: &http.Client{
			Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("network error")
			}),
		},
		baseURL: "https://mock.whatsapp.local",
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
	client := newMockClient(t, http.StatusOK, WhatsAppResponse{
		Messages: []struct {
			ID string `json:"id"`
		}{}, // Empty messages array
	}, nil)

	sender := &WhatsAppCloudSender{
		accessToken:   "test_token",
		phoneNumberID: "123456789",
		baseURL:       "https://mock.whatsapp.local",
		httpClient:    client,
	}

	_, err := sender.SendText("+2348001234567", "Test")
	if err == nil {
		t.Error("Expected error for missing message ID, got nil")
	}
}
