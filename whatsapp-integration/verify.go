package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
)

// verifyMetaSignature verifica la cabecera X-Hub-Signature-256 de Meta.
// Es crucial para asegurar que los webhooks provienen de Meta y no de un atacante.
func verifyMetaSignature(r *http.Request, bodyBytes []byte) error {
	// Solo verificar si la variable de entorno está configurada.
	appSecret := os.Getenv("META_APP_SECRET")
	if appSecret == "" {
		// En desarrollo, puede ser útil no requerir la firma.
		// En producción, esta variable SIEMPRE debe estar configurada.
		return nil
	}

	sigHeader := r.Header.Get("X-Hub-Signature-256")
	if sigHeader == "" {
		return fmt.Errorf("la cabecera X-Hub-Signature-256 no está presente")
	}

	// El formato de la cabecera es "sha256=FIRMA".
	parts := strings.SplitN(sigHeader, "=", 2)
	if len(parts) != 2 || parts[0] != "sha256" {
		return fmt.Errorf("formato de firma inválido")
	}

	// Calcular la firma esperada.
	mac := hmac.New(sha256.New, []byte(appSecret))
	mac.Write(bodyBytes)
	expectedMAC := mac.Sum(nil)
	expectedSignature := hex.EncodeToString(expectedMAC)

	// Comparar la firma recibida con la calculada.
	if !hmac.Equal([]byte(parts[1]), []byte(expectedSignature)) {
		return fmt.Errorf("las firmas no coinciden")
	}

	return nil
}
