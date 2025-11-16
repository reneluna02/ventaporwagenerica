package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"example.com/whatsapp-integration/adapter"
	"example.com/whatsapp-integration/bot"
	"example.com/whatsapp-integration/maps"
	"example.com/whatsapp-integration/store"
)

// IncomingMessage define la estructura del mensaje entrante.
type IncomingMessage struct {
	From string `json:"from"`
	Body string `json:"body"`
}

// WebhookPayload es la estructura del payload del webhook.
type WebhookPayload struct {
	Messages []IncomingMessage `json:"messages"`
}

func main() {
	// Configurar log
	logFile, err := os.OpenFile("whatsapp-integration.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		log.Fatalf("Error al abrir el archivo de log: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Configurar base de datos
	dbCfg := store.Config{
		Driver:   "sqlite3",
		Database: "whatsapp_bot.db",
	}
	dbStore, err := store.NewStore(dbCfg)
	if err != nil {
		log.Fatalf("Error inicializando la base de datos: %v", err)
	}
	defer dbStore.Close()

	// Configurar cliente de WhatsApp
	provider := os.Getenv("WHATSAPP_PROVIDER")
	if provider == "" {
		provider = "nodescript" // Predeterminado
	}
	waClient, err := adapter.NewClientFromEnv(provider)
	if err != nil {
		log.Printf("ADVERTENCIA: Cliente de WhatsApp no configurado (%v). Usando modo simulado.\n", err)
		waClient = adapter.NewMockClient()
	}

	// Configurar cliente de mapas
	mapsClient, err := maps.NewClient()
	if err != nil {
		log.Printf("ADVERTENCIA: Cliente de Maps no configurado (%v). La geocodificación no funcionará.\n", err)
	}

	// Iniciar el bot (máquina de estados)
	stateMachine := bot.NewStateMachine(dbStore, waClient, mapsClient)

	// Configurar rutas del servidor web
	http.HandleFunc("/webhook", webhookHandler(stateMachine))
	http.HandleFunc("/health", healthCheckHandler)

	// Iniciar servidor
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
		log.Fatalf("Error fatal del servidor: %v", err)
	}
}

func webhookHandler(bot *bot.StateMachine) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			handleVerification(w, r)
			return
		}

		if r.Method == http.MethodPost {
			handleWebhookPost(w, r, bot)
			return
		}

		http.Error(w, "Método no permitido", http.StatusMethodNotAllowed)
	}
}

func handleVerification(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	mode := q.Get("hub.mode")
	challenge := q.Get("hub.challenge")
	verifyToken := q.Get("hub.verify_token")

	if mode == "subscribe" && verifyToken == os.Getenv("WEBHOOK_VERIFY_TOKEN") {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(challenge))
	} else {
		http.Error(w, "Token de verificación inválido", http.StatusForbidden)
	}
}

func handleWebhookPost(w http.ResponseWriter, r *http.Request, bot *bot.StateMachine) {
	// Leer el cuerpo de la petición una sola vez.
	bodyBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error al leer el cuerpo de la petición: %v\n", err)
		http.Error(w, "Error interno", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	// Verificar la firma ANTES de procesar nada.
	if err := verifyMetaSignature(r, bodyBytes); err != nil {
		log.Printf("Firma del webhook inválida: %v\n", err)
		http.Error(w, "Firma inválida", http.StatusForbidden)
		return
	}

	// Ahora, decodificar el payload.
	var payload WebhookPayload
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		log.Printf("Error parseando JSON del webhook: %v\n", err)
		http.Error(w, "Cuerpo de la petición inválido", http.StatusBadRequest)
		return
	}

	if len(payload.Messages) == 0 || payload.Messages[0].Body == "" {
		w.WriteHeader(http.StatusOK)
		return
	}

	msg := payload.Messages[0]
	log.Printf("Recibido de %s: %s\n", msg.From, msg.Body)

	// Procesar el mensaje de forma asíncrona
	go func(from, body string) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if err := bot.ProcessMessage(ctx, from, body); err != nil {
			log.Printf("Error procesando mensaje para %s: %v\n", from, err)
		}
	}(msg.From, msg.Body)

	w.WriteHeader(http.StatusOK)
}

func healthCheckHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
