Ejemplo de integración mínima con WhatsApp mediante un script de Node.js

Resumen
------
Proyecto de ejemplo que expone un endpoint `/webhook` (GET para verificación, POST para recibir mensajes) y un adaptador para enviar mensajes a través de un script de Node.js.

El adaptador se configura por variables de entorno; si faltan, el proyecto cae a un modo 'mock' que solo loggea los envíos.

Cómo usar
-------
1) Copia el ejemplo y entra al directorio:

```bash
cd /tmp/whatsapp-integration
```

2) Llena las variables de entorno. Necesitarás definir la ruta a tu script de Node.js:

```bash
export WHATSAPP_PROVIDER=nodescript
export NODE_SCRIPT_PATH=/ruta/a/tu/script.js
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

Copia la URL pública que ngrok te da, añade `/webhook` y úsala para configurar el webhook en tu proveedor de WhatsApp.

Endpoints
--------
- GET /webhook?hub.mode=subscribe&hub.verify_token=TOKEN&hub.challenge=CH
  -> verificación (responde CH si el token coincide con `WEBHOOK_VERIFY_TOKEN`)
- POST /webhook
  -> recibe JSON con `messages: [{from, body}]` y responde 200. El servidor enviará una respuesta automática via el adaptador.

Notas
----
- Este ejemplo no incluye la lógica de negocio completa. Integra las llamadas a `adapter.NewClientFromEnv` y `SendMessage` con tu flujo original (`processMessage` en tu código).
- Los logs, incluyendo errores, se guardan en el archivo `whatsapp-integration.log`.
