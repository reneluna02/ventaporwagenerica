package adapter

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

// WhatsAppClient es la interfaz para enviar mensajes
type WhatsAppClient interface {
	SendMessage(to string, text string) error
}

// NewClientFromEnv crea el cliente según WHATSAPP_PROVIDER
func NewClientFromEnv(provider string) (WhatsAppClient, error) {
	switch strings.ToLower(provider) {
	case "nodescript":
		scriptPath := os.Getenv("NODE_SCRIPT_PATH")
		if scriptPath == "" {
			return nil, errors.New("la variable NODE_SCRIPT_PATH no está configurada")
		}
		return &NodeScriptClient{ScriptPath: scriptPath}, nil
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

func (m *MockClient) SendImage(to string, imageURL string, caption string) error {
	log.Printf("[MOCK] Enviando imagen a %s: %s (caption: %s)\n", to, imageURL, caption)
	return nil
}

// ----------------- Node.js Script Client -----------------

type NodeScriptClient struct {
	ScriptPath string
}

func (n *NodeScriptClient) SendMessage(to string, text string) error {
	if n.ScriptPath == "" {
		return errors.New("ruta del script de Node.js no configurada")
	}
	cmd := exec.Command("node", n.ScriptPath, to, text)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error ejecutando el script de Node.js: %v\nOutput: %s", err, string(output))
	}
	log.Printf("Script de Node.js ejecutado exitosamente para %s. Output: %s\n", to, string(output))
	return nil
}

func (n *NodeScriptClient) SendImage(to string, imageURL string, caption string) error {
	if n.ScriptPath == "" {
		return errors.New("ruta del script de Node.js no configurada")
	}
	// Asumimos que el script de Node.js puede manejar un argumento adicional para el caption.
	cmd := exec.Command("node", n.ScriptPath, to, imageURL, caption)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("error ejecutando el script de Node.js para enviar imagen: %v\nOutput: %s", err, string(output))
	}
	log.Printf("Script de Node.js para imagen ejecutado exitosamente para %s. Output: %s\n", to, string(output))
	return nil
}
