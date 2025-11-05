package adapter

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
)

// WhatsAppClient es la interfaz para enviar mensajes
type WhatsAppClient interface {
	SendMessage(to string, text string) error
}

// NewClientFromEnv crea el cliente según WHATSAPP_PROVIDER
func NewClientFromEnv(provider string) (WhatsAppClient, error) {
	switch strings.ToLower(provider) {
	case "twilio":
		acct := os.Getenv("TWILIO_ACCOUNT_SID")
		token := os.Getenv("TWILIO_AUTH_TOKEN")
		from := os.Getenv("TWILIO_FROM")
		if acct == "" || token == "" || from == "" {
			return nil, errors.New("variables TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN o TWILIO_FROM no están configuradas")
		}
		return &TwilioClient{AccountSID: acct, AuthToken: token, From: from}, nil
	case "meta":
		metaToken := os.Getenv("META_TOKEN")
		phoneID := os.Getenv("META_PHONE_NUMBER_ID")
		if metaToken == "" || phoneID == "" {
			return nil, errors.New("variables META_TOKEN o META_PHONE_NUMBER_ID no están configuradas")
		}
		return &MetaClient{AccessToken: metaToken, PhoneNumberID: phoneID}, nil
	default:
		return nil, fmt.Errorf("provider desconocido: %s", provider)
	}
}

// ----------------- Mock Client -----------------

type MockClient struct{}

func NewMockClient() *MockClient { return &MockClient{} }

func (m *MockClient) SendMessage(to string, text string) error {
	log.Printf("[MOCK] Enviando a %s: %s\n", to, text)
	return nil
}

// ----------------- Twilio Client -----------------

type TwilioClient struct {
	AccountSID string
	AuthToken  string
	From       string // Debe ser con prefijo 'whatsapp:' si usas sandbox
}

func (t *TwilioClient) SendMessage(to string, text string) error {
	api := fmt.Sprintf("https://api.twilio.com/2010-04-01/Accounts/%s/Messages.json", t.AccountSID)
	data := url.Values{}
	// Para WhatsApp via Twilio los números se formatean 'whatsapp:+521234...' según su documentación
	data.Set("To", ensureWhatsAppPrefix(to))
	data.Set("From", ensureWhatsAppPrefix(t.From))
	data.Set("Body", text)

	req, err := http.NewRequest("POST", api, strings.NewReader(data.Encode()))
	if err != nil { return err }
	req.SetBasicAuth(t.AccountSID, t.AuthToken)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("twilio API returned status %d", resp.StatusCode)
	}
	return nil
}

func ensureWhatsAppPrefix(n string) string {
	if strings.HasPrefix(n, "whatsapp:") {
		return n
	}
	if strings.HasPrefix(n, "+") {
		return "whatsapp:" + n
	}
	return n
}

// ----------------- Meta (WhatsApp Cloud) Client -----------------

type MetaClient struct {
	AccessToken   string
	PhoneNumberID string
}

func (m *MetaClient) SendMessage(to string, text string) error {
	api := fmt.Sprintf("https://graph.facebook.com/v15.0/%s/messages", m.PhoneNumberID)
	body := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                to,
		"type":              "text",
		"text": map[string]string{
			"body": text,
		},
	}
	b, _ := json.Marshal(body)
	req, err := http.NewRequest("POST", api, bytes.NewReader(b))
	if err != nil { return err }
	req.Header.Set("Authorization", "Bearer "+m.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil { return err }
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("meta graph API returned status %d", resp.StatusCode)
	}
	return nil
}
