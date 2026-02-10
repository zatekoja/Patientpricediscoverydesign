package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// WhatsAppCloudSender sends messages via WhatsApp Cloud API
type WhatsAppCloudSender struct {
	accessToken   string
	phoneNumberID string
	httpClient    *http.Client
	baseURL       string
}

// NewWhatsAppCloudSender creates a new WhatsApp sender
func NewWhatsAppCloudSender() (*WhatsAppCloudSender, error) {
	accessToken := os.Getenv("WHATSAPP_ACCESS_TOKEN")
	phoneNumberID := os.Getenv("WHATSAPP_PHONE_NUMBER_ID")

	if accessToken == "" || phoneNumberID == "" {
		return nil, fmt.Errorf("WHATSAPP_ACCESS_TOKEN and WHATSAPP_PHONE_NUMBER_ID must be set")
	}

	return &WhatsAppCloudSender{
		accessToken:   accessToken,
		phoneNumberID: phoneNumberID,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://graph.facebook.com/v18.0",
	}, nil
}

// WhatsAppTemplateMessage represents a template message
type WhatsAppTemplateMessage struct {
	MessagingProduct string                      `json:"messaging_product"`
	RecipientType    string                      `json:"recipient_type"`
	To               string                      `json:"to"`
	Type             string                      `json:"type"`
	Template         WhatsAppTemplateMessageBody `json:"template"`
}

// WhatsAppTemplateMessageBody represents the template body
type WhatsAppTemplateMessageBody struct {
	Name       string                             `json:"name"`
	Language   WhatsAppLanguage                   `json:"language"`
	Components []WhatsAppTemplateMessageComponent `json:"components,omitempty"`
}

// WhatsAppLanguage represents the language code
type WhatsAppLanguage struct {
	Code string `json:"code"`
}

// WhatsAppTemplateMessageComponent represents a template component
type WhatsAppTemplateMessageComponent struct {
	Type       string                             `json:"type"`
	Parameters []WhatsAppTemplateMessageParameter `json:"parameters"`
}

// WhatsAppTemplateMessageParameter represents a template parameter
type WhatsAppTemplateMessageParameter struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// WhatsAppTextMessage represents a text message
type WhatsAppTextMessage struct {
	MessagingProduct string `json:"messaging_product"`
	RecipientType    string `json:"recipient_type"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		PreviewURL bool   `json:"preview_url"`
		Body       string `json:"body"`
	} `json:"text"`
}

// WhatsAppResponse represents the API response
type WhatsAppResponse struct {
	MessagingProduct string `json:"messaging_product"`
	Contacts         []struct {
		Input string `json:"input"`
		WaID  string `json:"wa_id"`
	} `json:"contacts"`
	Messages []struct {
		ID string `json:"id"`
	} `json:"messages"`
}

// SendTemplate sends a template message
func (w *WhatsAppCloudSender) SendTemplate(to, templateName, languageCode string, parameters []string) (string, error) {
	// Build component parameters
	var components []WhatsAppTemplateMessageComponent
	if len(parameters) > 0 {
		params := make([]WhatsAppTemplateMessageParameter, len(parameters))
		for i, param := range parameters {
			params[i] = WhatsAppTemplateMessageParameter{
				Type: "text",
				Text: param,
			}
		}
		components = append(components, WhatsAppTemplateMessageComponent{
			Type:       "body",
			Parameters: params,
		})
	}

	message := WhatsAppTemplateMessage{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "template",
		Template: WhatsAppTemplateMessageBody{
			Name:       templateName,
			Language:   WhatsAppLanguage{Code: languageCode},
			Components: components,
		},
	}

	return w.sendMessage(message)
}

// SendText sends a text message
func (w *WhatsAppCloudSender) SendText(to, body string) (string, error) {
	message := WhatsAppTextMessage{
		MessagingProduct: "whatsapp",
		RecipientType:    "individual",
		To:               to,
		Type:             "text",
	}
	message.Text.PreviewURL = true
	message.Text.Body = body

	return w.sendMessage(message)
}

// sendMessage sends a message to WhatsApp Cloud API
func (w *WhatsAppCloudSender) sendMessage(message interface{}) (string, error) {
	url := fmt.Sprintf("%s/%s/messages", w.baseURL, w.phoneNumberID)

	jsonData, err := json.Marshal(message)
	if err != nil {
		return "", fmt.Errorf("failed to marshal message: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+w.accessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := w.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("WhatsApp API error (status %d): %s", resp.StatusCode, string(body))
	}

	var whatsappResp WhatsAppResponse
	if err := json.Unmarshal(body, &whatsappResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(whatsappResp.Messages) > 0 {
		return whatsappResp.Messages[0].ID, nil
	}

	return "", fmt.Errorf("no message ID in response")
}
