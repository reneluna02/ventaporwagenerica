package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"example.com/whatsapp-integration/adapter"
)

// Simple payload structures (varían según proveedor)
type IncomingMessage struct {
	From string `json:"from"`
	Body string `json:"body"`
}

type WebhookPayload struct {
	Messages []IncomingMessage `json:"messages"`
}

func main() {
	// Cargar configuración desde env
	provider := os.Getenv("WHATSAPP_PROVIDER") // "meta" o "twilio"
	if provider == "" {
		provider = "meta" // predeterminado
	}

	client, err := adapter.NewClientFromEnv(provider)
	if err != nil {
		log.Printf("Advertencia: cliente no configurado (%v). Se usará modo simulado.\n", err)
		client = adapter.NewMockClient()
	}

	// Rutas
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		// GET -> verificación simple (hub.challenge)
		if r.Method == http.MethodGet {
			q := r.URL.Query()
			mode := q.Get("hub.mode")
			challenge := q.Get("hub.challenge")
			verifyToken := q.Get("hub.verify_token")
			if mode == "subscribe" && verifyToken != "" {
				// Compara con env WEBHOOK_VERIFY_TOKEN
				if verifyToken == os.Getenv("WEBHOOK_VERIFY_TOKEN") {
					w.WriteHeader(http.StatusOK)
					w.Write([]byte(challenge))
					return
				}
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte("verify_token mismatch"))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// POST -> recibir mensajes
		if r.Method == http.MethodPost {
			// Limitar tamaño del body para seguridad
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MiB
			// Verificar firma (Meta) si está configurada
			if err := verifyMetaSignature(r); err != nil {
				log.Println("Firma del webhook inválida:", err)
				w.WriteHeader(http.StatusForbidden)
				return
			}

			var payload WebhookPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				log.Println("Error parseando JSON del webhook:", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if len(payload.Messages) == 0 {
				w.WriteHeader(http.StatusOK)
				return
			}

			msg := payload.Messages[0]
			log.Printf("Recibido de %s: %s\n", msg.From, msg.Body)

			// Procesar mensaje mínimo: responder con un eco y guía
			go func(from, body string) {
				resp := buildReply(body)
				if err := client.SendMessage(from, resp); err != nil {
					log.Println("Error enviando respuesta via WhatsApp client:", err)
				}
			}(msg.From, msg.Body)

			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusMethodNotAllowed)
	})

	// Health
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	log.Printf("Servidor escuchando en :%s (provider=%s)\n", port, provider)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

func buildReply(msg string) string {
	m := strings.TrimSpace(msg)
	upper := strings.ToUpper(m)
	if strings.Contains(upper, "HOLA") || strings.Contains(upper, "HI") {
		return "Hola! Gracias por escribir. Responde 1 para repetir pedido, 2 para nuevo pedido, 3 para registrarte."
	}
	return fmt.Sprintf("Recibimos: %s\n(Esto es un ejemplo. Integra tu lógica de negocio.)", m)
}
