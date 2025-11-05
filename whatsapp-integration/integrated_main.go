package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"

	"example.com/whatsapp-integration/adapter"
	"example.com/whatsapp-integration/bot"
	"example.com/whatsapp-integration/store"
)

type IncomingMessage struct {
	From string `json:"from"`
	Body string `json:"body"`
}

type WebhookPayload struct {
	Messages []IncomingMessage `json:"messages"`
}

func main() {
	_ = godotenv.Load()
	log.SetFormatter(&log.TextFormatter{FullTimestamp: true})
	logLevel, _ := log.ParseLevel(os.Getenv("LOG_LEVEL"))
	log.SetLevel(logLevel)

	// Configuración DB
	cfg := store.Config{
		Driver:   os.Getenv("DB_DRIVER"),
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		Database: os.Getenv("DB_NAME"),
	}
	st, err := store.NewStore(cfg)
	if err != nil {
		log.Fatalf("Error inicializando store: %v", err)
	}
	defer st.Close()

	// Adaptador WhatsApp
	provider := os.Getenv("WHATSAPP_PROVIDER")
	waClient, err := adapter.NewClientFromEnv(provider)
	if err != nil {
		log.Warnf("No se pudo inicializar adaptador WhatsApp: %v. Usando mock.", err)
		waClient = adapter.NewMockClient()
	}

	// Máquina de estados
	machine := bot.NewStateMachine(st, waClient)

	// Webhook
	http.HandleFunc("/webhook", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			q := r.URL.Query()
			mode := q.Get("hub.mode")
			challenge := q.Get("hub.challenge")
			verifyToken := q.Get("hub.verify_token")
			if mode == "subscribe" && verifyToken == os.Getenv("WEBHOOK_VERIFY_TOKEN") {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(challenge))
				return
			}
			w.WriteHeader(http.StatusForbidden)
			return
		}
		if r.Method == http.MethodPost {
			r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
			if err := verifyMetaSignature(r); err != nil {
				log.Warnf("Webhook firma inválida: %v", err)
				w.WriteHeader(http.StatusForbidden)
				return
			}
			var payload WebhookPayload
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				log.Warnf("Error parseando JSON: %v", err)
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if len(payload.Messages) == 0 {
				w.WriteHeader(http.StatusOK)
				return
			}
			msg := payload.Messages[0]
			log.Infof("Mensaje recibido de %s: %s", msg.From, msg.Body)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := machine.ProcessMessage(ctx, msg.From, msg.Body); err != nil {
				log.Errorf("Error procesando mensaje: %v", err)
			}
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
	log.Infof("Servidor escuchando en :%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}