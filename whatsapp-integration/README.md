Ejemplo de integración mínima con WhatsApp (Meta Cloud API o Twilio)

Resumen
------
Proyecto de ejemplo que expone un endpoint `/webhook` (GET para verificación, POST para recibir mensajes) y un adaptador para enviar mensajes a través de:

- Meta WhatsApp Cloud API (recommended)
- Twilio WhatsApp (opcional)

El adaptador se configura por variables de entorno; si faltan, el proyecto cae a un modo 'mock' que solo loggea los envíos.

Cómo usar
-------
1) Copia el ejemplo y entra al directorio:

```bash
cd /tmp/whatsapp-integration
```

2) Llena las variables de entorno (usa `.env.example` como guía) y exporta las que necesites. Ejemplo para Meta:

```bash
export WHATSAPP_PROVIDER=meta
export META_TOKEN=EAA... (tu token)
export META_PHONE_NUMBER_ID=1234567890
export WEBHOOK_VERIFY_TOKEN=mi_token_de_verificacion
```c

Para Twilio:

```bash
export WHATSAPP_PROVIDER=twilio
export TWILIO_ACCOUNT_SID=AC... 
export TWILIO_AUTH_TOKEN=your_auth_token
export TWILIO_FROM=whatsapp:+1415...  # tu número de Twilio (whatsapp:...) 
export WEBHOOK_VERIFY_TOKEN=mi_token_de_verificacion
```

3) Ejecuta:

```bash
go run ./...
```

4) Exponer localmente con ngrok para desarrollo y probar webhook:

```bash
ngrok http 8080
```

Copia la URL pública que ngrok te da, añade `/webhook` y úsala para configurar el webhook en Meta o Twilio.

Endpoints
--------
- GET /webhook?hub.mode=subscribe&hub.verify_token=TOKEN&hub.challenge=CH
  -> verificación (responde CH si el token coincide con `WEBHOOK_VERIFY_TOKEN`)
- POST /webhook
  -> recibe JSON con `messages: [{from, body}]` y responde 200. El servidor enviará una respuesta automática via el adaptador.

Notas
----
- Este ejemplo no incluye la lógica de negocio completa. Integra las llamadas a `adapter.NewClientFromEnv` y `SendMessage` con tu flujo original (`processMessage` en tu código).
- Para producción: valida signatures (X-Hub-Signature-256 para Meta), usa HTTPS, y guarda credenciales de forma segura.

Verificación de firma (Meta)
---------------------------
Si configuras `META_APP_SECRET` el servidor validará la cabecera `X-Hub-Signature-256` de Meta antes de procesar el POST.

Ejemplo de cómo generar la cabecera para probar con `curl` (Linux/macOS, usando OpenSSL):

```bash
# body.json contiene el payload que enviarás al webhook
body='{"messages":[{"from":"+521111222333","body":"hola"}]}'
export META_APP_SECRET="tu_app_secret"
sig=$(printf "%s" "$body" | openssl dgst -sha256 -hmac "$META_APP_SECRET" -binary | xxd -p -c 256)

curl -v -X POST \
  -H "Content-Type: application/json" \
  -H "X-Hub-Signature-256: sha256=$sig" \
  --data "$body" \
  http://localhost:8080/webhook
```

Si `META_APP_SECRET` no está seteado, el servidor no exige la firma (modo práctico para desarrollo). Para producción siempre define `META_APP_SECRET`.
