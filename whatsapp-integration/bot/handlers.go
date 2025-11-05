package bot

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"example.com/whatsapp-integration/store"
)

// handleInicial gestiona el primer mensaje o reset
func (sm *StateMachine) handleInicial(ctx context.Context, telefono string) error {
	pedido, err := sm.store.GetUltimoPedido(ctx, sm.session.ClienteActual.ID)
	if err != nil {
		return err
	}

	var msg string
	if pedido != nil {
		// Calcular precio actual
		precioActual := 12.50 // TODO: obtener de DB
		total := pedido.CantidadLitros * precioActual

		msg = fmt.Sprintf(
			"¡Hola %s %s!\n\n"+
				"Elige una opción:\n\n"+
				"1. Tu pedido será igual que el anterior:\n"+
				"   • %s\n"+
				"   • %.0f Lts\n"+
				"   • %s\n"+
				"   • *Precio actual: $%.2f*\n\n"+
				"2. Nuevo pedido (mismo domicilio)\n"+
				"3. Actualizar mis datos",
			sm.session.ClienteActual.Nombre,
			sm.session.ClienteActual.ApellidoPaterno,
			pedido.TipoServicio,
			pedido.CantidadLitros,
			pedido.Direccion,
			total)
	} else {
		msg = "¡Bienvenido! Por favor elige:\n\n" +
			"1. Nuevo pedido\n" +
			"2. Registrar mis datos"
	}

	if err := sm.sender.SendMessage(telefono, msg); err != nil {
		return err
	}
	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoOpcion)
}

// handleOpcionInicial procesa la selección inicial del cliente
func (sm *StateMachine) handleOpcionInicial(ctx context.Context, telefono, mensaje string) error {
	switch mensaje {
	case "1":
		// Repetir último pedido
		pedido, err := sm.store.GetUltimoPedido(ctx, sm.session.ClienteActual.ID)
		if err != nil {
			return err
		}
		if pedido == nil {
			return sm.handleTipoServicio(ctx, telefono, mensaje)
		}

		// Crear nuevo pedido basado en el anterior
		nuevoPedido := &store.Pedido{
			ClienteID:      pedido.ClienteID,
			TipoServicio:   pedido.TipoServicio,
			CantidadLitros: pedido.CantidadLitros,
			Direccion:      pedido.Direccion,
			ColorPuerta:    pedido.ColorPuerta,
			ColorFachada:   pedido.ColorFachada,
			Estado:         "pendiente",
		}

		if err := sm.store.CrearPedido(ctx, nuevoPedido); err != nil {
			return err
		}

		msg := "¡Pedido confirmado!\n\n" +
			"Un operador se comunicará contigo pronto.\n" +
			"También recibirás notificaciones del estado de tu pedido por este medio."

		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoInicial)

	case "2":
		// Nuevo pedido
		msg := "¿Tu servicio será para Tanque Estacionario o Cilindro?"
		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoTipo)

	case "3":
		// Actualizar datos
		msg := "Por favor, escribe tu nombre completo empezando por:\n" +
			"APELLIDO PATERNO APELLIDO MATERNO NOMBRE(S)\n\n" +
			"Ejemplo: González García Juan"
		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoNombre)

	default:
		msg := "Por favor selecciona una opción válida:\n" +
			"1. Repetir último pedido\n" +
			"2. Nuevo pedido\n" +
			"3. Actualizar datos"
		return sm.sender.SendMessage(telefono, msg)
	}
}

// handleNombre procesa el registro/actualización del nombre
func (sm *StateMachine) handleNombre(ctx context.Context, telefono, mensaje string) error {
	partes := strings.Split(strings.TrimSpace(mensaje), " ")
	if len(partes) < 3 {
		return sm.sender.SendMessage(telefono,
			"Por favor ingresa tu nombre completo en el formato:\n"+
				"APELLIDO PATERNO APELLIDO MATERNO NOMBRE(S)")
	}

	sm.session.ClienteActual.ApellidoPaterno = partes[0]
	sm.session.ClienteActual.ApellidoMaterno = partes[1]
	sm.session.ClienteActual.Nombre = strings.Join(partes[2:], " ")

	if err := sm.store.ActualizarCliente(ctx, sm.session.ClienteActual); err != nil {
		return err
	}

	msg := fmt.Sprintf("¡Gracias %s!\n\n"+
		"¿Tu servicio será para Tanque Estacionario o Cilindro?",
		sm.session.ClienteActual.Nombre)

	sm.sender.SendMessage(telefono, msg)
	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoTipo)
}

// handleFotoCasa procesa la respuesta inicial cuando se crea un cliente nuevo
// Opciones: 1 = Sí (enviar foto), 2 = No (describir colores)
func (sm *StateMachine) handleFotoCasa(ctx context.Context, telefono, mensaje string) error {
	switch strings.ToUpper(strings.TrimSpace(mensaje)) {
	case "1", "SI", "SÍ":
		// Pedir que envíen la foto y luego confirmar
		msg := "Por favor, envía una foto de tu casa ahora.\n\nCuando la envíes, responde 'Listo' y te preguntaremos si es la casa correcta."
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoConfirmandoFotoCasa)

	case "2", "NO":
		// Pedir colores
		msg := "Entendido. Para ayudar al repartidor, por favor indica:\n• Color de la puerta\n• Color de la fachada\n\nEjemplo: Puerta café, fachada blanca"
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		// Marcar cliente como "código rojo" para atención especial
		// (se asume que la estructura Cliente y el store pueden persistir este flag)
		if sm.session.ClienteActual != nil {
			// Intentar marcar y persistir (campo hipotético CodigoRojo)
			// Se intenta a nivel de cliente para que afecte futuros pedidos
			// Nota: si el store no tiene el campo, esta llamada puede necesitar adaptación
			sm.session.ClienteActual.CodigoRojo = true
			sm.store.ActualizarCliente(ctx, sm.session.ClienteActual)
		}
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoColorFachada)

	default:
		return sm.sender.SendMessage(telefono, "Por favor responde:\n1. Sí (enviar foto)\n2. No (describir colores)")
	}
}

// handleConfirmacionFotoCasa procesa la confirmación después de la foto
// Se espera que el cliente confirme si la foto enviada es su casa
func (sm *StateMachine) handleConfirmacionFotoCasa(ctx context.Context, telefono, mensaje string) error {
	switch strings.ToUpper(strings.TrimSpace(mensaje)) {
	case "LISTO":
		// Usuario indicó que subió la foto y está listo para confirmar
		msg := "¿Es esta tu casa?\n1. Sí\n2. No"
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoConfirmandoFotoCasa)

	case "1", "SI", "SÍ":
		// Confirmado: marcar código rojo y guardar
		if sm.session.ClienteActual != nil {
			sm.session.ClienteActual.CodigoRojo = true
			sm.store.ActualizarCliente(ctx, sm.session.ClienteActual)
		}
		msg := "¡Gracias! Hemos registrado la foto y marcado tu cuenta para atención especial (código rojo)."
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoInicial)

	case "2", "NO":
		// No es la casa: pedir colores
		msg := "Entendido. Por favor indica el color de la puerta y de la fachada.\nEjemplo: Puerta café, fachada blanca"
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		if sm.session.ClienteActual != nil {
			sm.session.ClienteActual.CodigoRojo = true
			sm.store.ActualizarCliente(ctx, sm.session.ClienteActual)
		}
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoColorFachada)

	default:
		return sm.sender.SendMessage(telefono, "Por favor responde:\n1. Sí\n2. No\nO envía 'Listo' cuando hayas subido la foto.")
	}
}

// handleTipoServicio procesa la selección del tipo de servicio
func (sm *StateMachine) handleTipoServicio(ctx context.Context, telefono, mensaje string) error {
	switch strings.ToUpper(mensaje) {
	case "ESTACIONARIO", "E":
		precioLitro := 12.50 // TODO: obtener de DB
		msg := fmt.Sprintf(
			"💧 *Tanque Estacionario*\n\n"+
				"Precio por litro: $%.2f\n\n"+
				"¿Cómo deseas hacer tu pedido?\n\n"+
				"1. Por Litros (ej: 150 Lts)\n"+
				"2. Por Dinero (ej: $500)\n"+
				"3. Usar Tabulador (para calcular llenado al 85%%)",
			precioLitro)

		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioMenu)

	case "CILINDRO", "C":
		msg := "🛢️ *Servicio de Cilindro*\n\n" +
			"¿Qué opción prefieres?\n\n" +
			"1. Recarga\n" +
			"   • Recogemos tu tanque\n" +
			"   • Asignamos código QR único\n" +
			"   • Te notificamos al recogerlo\n" +
			"   • Lo regresamos recargado\n\n" +
			"2. Canje\n" +
			"   • Te damos uno nuevo\n" +
			"   • Nos llevamos el tuyo"

		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoCilindroOpcion)

	default:
		return sm.sender.SendMessage(telefono,
			"Por favor escribe 'Estacionario' o 'Cilindro'")
	}
}

// handleEstacionarioMenu maneja el menú de opciones para tanque estacionario
func (sm *StateMachine) handleEstacionarioMenu(ctx context.Context, telefono, mensaje string) error {
	switch strings.ToUpper(mensaje) {
	case "1", "LITROS", "LTS":
		msg := "¿Cuántos litros necesitas?\n" +
			"(escribe solo el número, ejemplo: 150)"
		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioLts)

	case "2", "DINERO", "$":
		msg := "¿Cuánto quieres cargar en pesos?\n" +
			"(escribe solo el número, ejemplo: 500)"
		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioDinero)

	case "3", "TABULADOR":
		msg := "📊 *Tabulador de Llenado*\n\n" +
			"¿Cuál es la capacidad TOTAL de tu tanque en litros?\n" +
			"(escribe solo el número, ejemplo: 300)"
		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioTabuladorCapacidad)

	default:
		msg := "Por favor selecciona una opción válida:\n" +
			"1. Por Litros\n" +
			"2. Por Dinero\n" +
			"3. Usar Tabulador"
		return sm.sender.SendMessage(telefono, msg)
	}
}

// handleEstacionarioLitros procesa pedido por litros
func (sm *StateMachine) handleEstacionarioLitros(ctx context.Context, telefono, mensaje string) error {
	litros, err := strconv.ParseFloat(mensaje, 64)
	if err != nil {
		return sm.sender.SendMessage(telefono,
			"Por favor ingresa solo números (ejemplo: 150)")
	}

	precioLitro := 12.50 // TODO: obtener de DB
	total := litros * precioLitro

	msg := fmt.Sprintf(
		"📝 *Resumen del Pedido*\n\n"+
			"• Cantidad: %.0f litros\n"+
			"• Precio: $%.2f/litro\n"+
			"• Total: $%.2f\n\n"+
			"¿Cómo deseas pagar?\n"+
			"1. Efectivo\n"+
			"2. Tarjeta (terminal)",
		litros, precioLitro, total)

	// Guardar datos del pedido en sesión
	sm.session.PedidoEnCurso = &store.Pedido{
		ClienteID:      sm.session.ClienteActual.ID,
		TipoServicio:   "estacionario",
		CantidadLitros: litros,
		PrecioUnitario: precioLitro,
	}

	sm.sender.SendMessage(telefono, msg)
	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoPago)
}

// handleEstacionarioDinero procesa pedido por monto
func (sm *StateMachine) handleEstacionarioDinero(ctx context.Context, telefono, mensaje string) error {
	monto, err := strconv.ParseFloat(strings.TrimPrefix(mensaje, "$"), 64)
	if err != nil {
		return sm.sender.SendMessage(telefono,
			"Por favor ingresa solo números (ejemplo: 500)")
	}

	precioLitro := 12.50 // TODO: obtener de DB
	litros := monto / precioLitro

	msg := fmt.Sprintf(
		"📝 *Resumen del Pedido*\n\n"+
			"• Monto: $%.2f\n"+
			"• Precio: $%.2f/litro\n"+
			"• Cantidad: %.1f litros\n\n"+
			"¿Cómo deseas pagar?\n"+
			"1. Efectivo\n"+
			"2. Tarjeta (terminal)",
		monto, precioLitro, litros)

	sm.session.PedidoEnCurso = &store.Pedido{
		ClienteID:      sm.session.ClienteActual.ID,
		TipoServicio:   "estacionario",
		CantidadLitros: litros,
		CantidadDinero: monto,
		PrecioUnitario: precioLitro,
	}

	sm.sender.SendMessage(telefono, msg)
	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoPago)
}

// handlePago procesa la selección del método de pago
func (sm *StateMachine) handlePago(ctx context.Context, telefono, mensaje string) error {
	switch strings.ToUpper(mensaje) {
	case "1", "EFECTIVO":
		sm.session.PedidoEnCurso.MetodoPago = "efectivo"
	case "2", "TARJETA":
		sm.session.PedidoEnCurso.MetodoPago = "tarjeta"
	default:
		return sm.sender.SendMessage(telefono,
			"Por favor selecciona:\n1. Efectivo\n2. Tarjeta")
	}

	msg := "Por favor, escribe tu dirección completa incluyendo:\n" +
		"• Calle y número\n" +
		"• Colonia\n" +
		"• Referencias\n\n" +
		"Ejemplo: Av. Siempre Viva 123, Springfield, junto a la tienda"

	sm.sender.SendMessage(telefono, msg)
	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoDireccion)
}

// handleDireccion procesa y verifica la dirección
func (sm *StateMachine) handleDireccion(ctx context.Context, telefono, direccion string) error {
	sm.session.PedidoEnCurso.Direccion = direccion

	// TODO: Integración real con Maps API
	mapsURL := fmt.Sprintf("https://maps.google.com/?q=%s", url.QueryEscape(direccion))
	streetViewURL := "https://maps.google.com/streetview..."

	sm.sender.SendMessage(telefono, "📍 *Ubicación*\n\n"+mapsURL)

	msg := "🏠 *¿Es esta tu casa?*\n\n" +
		"1. Sí, es correcta\n" +
		"2. No, necesito especificar más"

	sm.sender.SendMessage(telefono, msg)
	return sm.actualizarEstado(ctx, telefono, EstadoConfirmandoDireccion)
}

// handleConfirmacionDireccion procesa la confirmación de la dirección
func (sm *StateMachine) handleConfirmacionDireccion(ctx context.Context, telefono, mensaje string) error {
	switch strings.ToUpper(mensaje) {
	case "1", "SI", "SÍ":
		return sm.finalizarPedido(ctx, telefono)

	case "2", "NO":
		msg := "Para ayudar al repartidor, por favor indica:\n" +
			"• Color de la puerta\n" +
			"• Color de la fachada\n\n" +
			"Ejemplo: Puerta café, fachada blanca"

		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoColorFachada)

	default:
		return sm.sender.SendMessage(telefono,
			"Por favor responde:\n1. Sí\n2. No")
	}
}

// handleColorFachada procesa los colores de la casa
func (sm *StateMachine) handleColorFachada(ctx context.Context, telefono, mensaje string) error {
	// Guardar colores bien sea en el pedido en curso o en el perfil del cliente
	if sm.session.PedidoEnCurso != nil {
		sm.session.PedidoEnCurso.ColorPuerta = mensaje // TODO: parsear mejor
		sm.session.PedidoEnCurso.ColorFachada = mensaje
		// Marcar pedido especial (código rojo) si aplica
		sm.session.PedidoEnCurso.CodigoRojo = true
		return sm.finalizarPedido(ctx, telefono)
	}

	// Si no hay pedido en curso — estamos en flujo de registro — guardar en cliente
	if sm.session.ClienteActual != nil {
		// Suponemos que Cliente tiene campos ColorPuerta/ColorFachada/CodigoRojo
		sm.session.ClienteActual.ColorPuerta = mensaje
		sm.session.ClienteActual.ColorFachada = mensaje
		sm.session.ClienteActual.CodigoRojo = true
		if err := sm.store.ActualizarCliente(ctx, sm.session.ClienteActual); err != nil {
			return err
		}
		msg := "¡Listo! Hemos guardado los colores y marcado tu cuenta para atención especial (código rojo)."
		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoInicial)
	}

	// Fallback: pedir que reinicien el flujo
	return sm.sender.SendMessage(telefono, "No se pudo guardar la información. Por favor intenta de nuevo.")
}

// finalizarPedido guarda y confirma el pedido
func (sm *StateMachine) finalizarPedido(ctx context.Context, telefono string) error {
	if err := sm.store.CrearPedido(ctx, sm.session.PedidoEnCurso); err != nil {
		return err
	}

	msg := "🎉 *¡Pedido Confirmado!*\n\n" +
		"Tu pedido ha sido registrado y está en proceso.\n" +
		"Recibirás actualizaciones por este medio.\n\n" +
		"Gracias por tu preferencia."

	sm.sender.SendMessage(telefono, msg)
	return sm.actualizarEstado(ctx, telefono, EstadoInicial)
}

// handleReporteSello procesa reportes de sello violado
func (sm *StateMachine) handleReporteSello(ctx context.Context, telefono string) error {
	reporte := &store.ReporteSello{
		ClienteID:   sm.session.ClienteActual.ID,
		PedidoID:    sm.session.PedidoEnCurso.ID,
		TipoReporte: "sello_violado",
		Estado:      "pendiente",
		Descripcion: "Reporte de sello violado",
	}

	if err := sm.store.CrearReporteSello(ctx, reporte); err != nil {
		return err
	}

	msg := "⚠️ *Reporte Recibido*\n\n" +
		"Tu reporte ha sido registrado.\n" +
		"Un supervisor revisará el caso inmediatamente.\n" +
		"Por favor conserva el tanque en su estado actual.\n\n" +
		"¿Deseas enviar una foto del sello?\n" +
		"1. Sí\n2. No"

	sm.sender.SendMessage(telefono, msg)
	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoFotoSello)
}
