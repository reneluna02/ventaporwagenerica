package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// verifyMetaSignature verifica la cabecera X-Hub-Signature-256 usando META_APP_SECRET
// Si no está configurada la variable de entorno, la función devuelve nil (no obliga la verificación).
func verifyMetaSignature(r *http.Request) error {
	header := r.Header.Get("X-Hub-Signature-256")
	// Si no hay header y no hay secreto configurado, no forzamos la verificación
	secret := os.Getenv("META_APP_SECRET")
	if secret == "" {
		// No hay secreto configurado: omitimos la verificación (modo dev)
		return nil
	}
	if header == "" {
		return fmt.Errorf("cabecera X-Hub-Signature-256 ausente")
	}
	parts := strings.SplitN(header, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("formato de cabecera inválido")
	}
	expectedHex := parts[1]

	// Leer body completo
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("no se pudo leer body: %w", err)
	}
	// Restaurar el body para que pueda ser leído posteriormente
	r.Body = io.NopCloser(bytes.NewReader(body))

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	sum := mac.Sum(nil)
	computedHex := hex.EncodeToString(sum)

	// Comparación en tiempo constante
	if subtle.ConstantTimeCompare([]byte(computedHex), []byte(expectedHex)) != 1 {
		return fmt.Errorf("firma no coincide")
	}
	return nil
}
