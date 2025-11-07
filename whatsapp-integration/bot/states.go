package bot

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"example.com/whatsapp-integration/store"
)

// Estados de la conversación
const (
	// Estados iniciales
	EstadoInicial           = "INICIO"
	EstadoEsperandoOpcion   = "ESPERANDO_OPCION_INICIAL"
	EstadoEsperandoNombre   = "ESPERANDO_NOMBRE_NUEVO"

	// Estado para primer registro: foto de la casa
	EstadoEsperandoFotoCasa     = "ESPERANDO_FOTO_CASA"      // Pregunta inicial: ¿puedes enviar foto? (1=Sí 2=No)
	EstadoConfirmandoFotoCasa   = "CONFIRMANDO_FOTO_CASA"    // Después de recibir foto, confirmar si es la casa

	// Estados para tipo de servicio
	EstadoEsperandoTipo     = "ESPERANDO_TIPO_SERVICIO"

	// Estados para estacionario
	EstadoEstacionarioMenu        = "ESPERANDO_OPCION_ESTACIONARIO" // Litros, Dinero o Tabulador
	EstadoEstacionarioLts        = "ESPERANDO_LITROS_ESTACIONARIO"
	EstadoEstacionarioDinero     = "ESPERANDO_DINERO_ESTACIONARIO"
	EstadoEstacionarioTabuladorCapacidad = "ESPERANDO_CAPACIDAD_TABULADOR"
	EstadoEstacionarioTabuladorPorcentaje = "ESPERANDO_PORCENTAJE_TABULADOR"
	EstadoEstacionarioConfirmacion = "CONFIRMANDO_PEDIDO_ESTACIONARIO"

	// Estados para cilindro
	EstadoCilindroOpcion         = "ESPERANDO_OPCION_CILINDRO"      // Recarga o Canje
	EstadoCilindroCantidad       = "ESPERANDO_CANTIDAD_CILINDRO"
	EstadoCilindroConfirmacionQR = "CONFIRMANDO_QR_CILINDRO"        // Cliente confirma QR
	EstadoCilindroRecoleccion    = "ESPERANDO_RECOLECCION"          // Esperando que operador recoja
	EstadoCilindroEntrega        = "ESPERANDO_ENTREGA"              // En ruta de regreso

	// Estados de pago y dirección
	EstadoEsperandoPago         = "ESPERANDO_METODO_PAGO"
	EstadoEsperandoDireccion    = "ESPERANDO_DIRECCION"
	EstadoConfirmandoDireccion  = "CONFIRMANDO_DIRECCION"          // Con Maps/Street View
	EstadoConfirmandoPedidoFinal = "CONFIRMANDO_PEDIDO_FINAL"
	EstadoEsperandoColorFachada = "ESPERANDO_COLOR_FACHADA"

	// Estados especiales
	EstadoReportandoSello      = "REPORTANDO_SELLO"               // Cliente reporta sello violado
	EstadoEsperandoFotoSello   = "ESPERANDO_FOTO_SELLO"          // Opcional: foto del sello
	EstadoConfirmandoEntrega   = "CONFIRMANDO_ENTREGA"           // Cliente confirma recepción
)

// WhatsAppSender es una interfaz para enviar mensajes
type WhatsAppSender interface {
	SendMessage(to string, text string) error
}

// StateMachine maneja la lógica de estados del bot
type StateMachine struct {
	store   store.Store
	sender  WhatsAppSender
	session *Session // mantiene datos temporales entre estados
}

// Session mantiene datos temporales entre estados
type Session struct {
	ClienteActual  *store.Cliente
	PedidoEnCurso  *store.Pedido
	DatosTemp      map[string]interface{}
}

func NewStateMachine(s store.Store, sender WhatsAppSender) *StateMachine {
	return &StateMachine{
		store:  s,
		sender: sender,
		session: &Session{
			DatosTemp: make(map[string]interface{}),
		},
	}
}

// ProcessMessage procesa un mensaje entrante según el estado actual
func (sm *StateMachine) ProcessMessage(ctx context.Context, telefono, mensaje string) error {
	// Verificar reporte de sello (interrumpe flujo normal)
	if strings.Contains(strings.ToUpper(mensaje), "REPORTAR SELLO") {
		return sm.handleReporteSello(ctx, telefono)
	}

	// Buscar o crear cliente
	cliente, err := sm.store.GetClientePorTelefono(ctx, telefono)
	if err != nil {
		return fmt.Errorf("error buscando cliente: %w", err)
	}
	if cliente == nil {
		// Nuevo cliente: solicitar el nombre.
		cliente = &store.Cliente{
			NumeroTelefono:    telefono,
			EstadoConversacion: EstadoEsperandoNombre,
		}
		if err := sm.store.CrearCliente(ctx, cliente); err != nil {
			return fmt.Errorf("error creando cliente: %w", err)
		}

		sm.session.ClienteActual = cliente
		msg := "¡Bienvenido! Para registrarte, por favor escribe tu nombre completo, empezando por tu apellido paterno. Ejemplo: Pérez López Juan."
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return fmt.Errorf("error enviando saludo a nuevo cliente: %w", err)
		}
		// El estado ya está puesto, solo queda esperar la respuesta del usuario.
		return nil
	}

	sm.session.ClienteActual = cliente
	return sm.handleState(ctx, telefono, mensaje, cliente.EstadoConversacion)
}

func (sm *StateMachine) handleState(ctx context.Context, telefono, mensaje, estado string) error {
	var err error
	switch estado {
	case EstadoInicial:
		err = sm.handleInicial(ctx, telefono)
	
	case EstadoEsperandoOpcion:
		err = sm.handleOpcionInicial(ctx, telefono, mensaje)
	
	case EstadoEsperandoNombre:
		err = sm.handleNombre(ctx, telefono, mensaje)

	case EstadoEsperandoFotoCasa:
		err = sm.handleFotoCasa(ctx, telefono, mensaje)

	case EstadoConfirmandoFotoCasa:
		err = sm.handleConfirmacionFotoCasa(ctx, telefono, mensaje)
	
	case EstadoEsperandoTipo:
		err = sm.handleTipoServicio(ctx, telefono, mensaje)
	
	case EstadoEstacionarioMenu:
		err = sm.handleEstacionarioMenu(ctx, telefono, mensaje)
	
	case EstadoEstacionarioLts:
		err = sm.handleEstacionarioLitros(ctx, telefono, mensaje)
	
	case EstadoEstacionarioDinero:
		err = sm.handleEstacionarioDinero(ctx, telefono, mensaje)
	
	case EstadoEstacionarioTabuladorCapacidad:
		err = sm.handleTabuladorCapacidad(ctx, telefono, mensaje)
	
	case EstadoEstacionarioTabuladorPorcentaje:
		err = sm.handleTabuladorPorcentaje(ctx, telefono, mensaje)

	case EstadoEstacionarioConfirmacion:
		err = sm.handleEstacionarioConfirmacion(ctx, telefono, mensaje)
	
	case EstadoCilindroOpcion:
		err = sm.handleCilindroOpcion(ctx, telefono, mensaje)
	
	case EstadoCilindroCantidad:
		err = sm.handleCilindroCantidad(ctx, telefono, mensaje)
	
	case EstadoCilindroConfirmacionQR:
		err = sm.handleConfirmacionQR(ctx, telefono, mensaje)
	
	case EstadoEsperandoPago:
		err = sm.handlePago(ctx, telefono, mensaje)
	
	case EstadoEsperandoDireccion:
		err = sm.handleDireccion(ctx, telefono, mensaje)
	
	case EstadoConfirmandoDireccion:
		err = sm.handleConfirmacionDireccion(ctx, telefono, mensaje)

	case EstadoConfirmandoPedidoFinal:
		err = sm.handleConfirmacionFinalPost(ctx, telefono, mensaje)
	
	case EstadoEsperandoColorFachada:
		err = sm.handleColorFachada(ctx, telefono, mensaje)
	
	case EstadoReportandoSello:
		err = sm.handleReporteSello(ctx, telefono)
	
	case EstadoEsperandoFotoSello:
		err = sm.handleFotoSello(ctx, telefono, mensaje)
	
	case EstadoConfirmandoEntrega:
		err = sm.handleConfirmacionEntrega(ctx, telefono, mensaje)
		
	default:
		err = fmt.Errorf("estado no manejado: %s", estado)
	}

	if err != nil {
		// Log del error pero continuamos
		fmt.Printf("Error manejando estado %s: %v\n", estado, err)
		sm.sender.SendMessage(telefono, "Hubo un error procesando tu mensaje. Por favor intenta de nuevo.")
		return err
	}
	return nil
}

func (sm *StateMachine) handleInicial(ctx context.Context, telefono string) error {
	pedido, err := sm.store.GetUltimoPedido(ctx, sm.session.ClienteActual.ID)
	if err != nil {
		return err
	}

	var msg string
	if pedido != nil {
		msg = fmt.Sprintf("¡Hola %s!\n\nElige una opción:\n\n1. Repetir pedido anterior:\n   - %s\n   - %.0f Lts\n   - %s\n\n2. Nuevo pedido (mismo domicilio)\n3. Actualizar datos",
			sm.session.ClienteActual.Nombre,
			pedido.TipoServicio,
			pedido.CantidadLitros,
			pedido.Direccion)
	} else {
		msg = fmt.Sprintf("¡Hola %s! Veo que aún no tienes pedidos con nosotros.\n\nElige una opción:\n\n1. Hacer un nuevo pedido\n2. Actualizar mis datos", sm.session.ClienteActual.Nombre)
	}

	if err := sm.sender.SendMessage(telefono, msg); err != nil {
		return err
	}

	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoOpcion)
}

func (sm *StateMachine) handleOpcionInicial(ctx context.Context, telefono, mensaje string) error {
	pedido, err := sm.store.GetUltimoPedido(ctx, sm.session.ClienteActual.ID)
	if err != nil {
		return fmt.Errorf("error al obtener último pedido: %w", err)
	}

	// Flujo para clientes SIN pedidos previos
	if pedido == nil {
		switch mensaje {
		case "1": // Hacer un nuevo pedido
			return sm.handleTipoServicio(ctx, telefono, mensaje)
		case "2": // Actualizar mis datos
			sm.sender.SendMessage(telefono, "Por favor, escribe tu nombre completo (Apellido Paterno, Apellido Materno, Nombre)")
			return sm.actualizarEstado(ctx, telefono, EstadoEsperandoNombre)
		default:
			sm.sender.SendMessage(telefono, "Opción no válida. Por favor elige 1 o 2.")
			return nil
		}
	}

	// Flujo para clientes CON pedidos previos
	switch mensaje {
	case "1": // Repetir pedido
		precioActualLitro := 12.50 // Simulación de precio actual.
		nuevoPedido := *pedido
		nuevoPedido.ID = 0 // Es un nuevo registro en la BD.
		nuevoPedido.Estado = "pendiente"
		nuevoPedido.PrecioUnitario = precioActualLitro
		nuevoPedido.CantidadDinero = nuevoPedido.CantidadLitros * precioActualLitro // Recalcular total.

		if err := sm.store.CrearPedido(ctx, &nuevoPedido); err != nil {
			return err
		}

		msg := fmt.Sprintf(
			"✅ *Pedido Confirmado*\n\n"+
				"Hemos registrado la repetición de tu último pedido con los precios actualizados:\n\n"+
				"  - *Servicio:* %s\n"+
				"  - *Cantidad:* %.0f Lts\n"+
				"  - *Precio por Litro:* $%.2f\n"+
				"  - *Total a Pagar:* $%.2f\n\n"+
				"En breve, nuestro equipo te confirmará la entrega.",
			nuevoPedido.TipoServicio,
			nuevoPedido.CantidadLitros,
			nuevoPedido.PrecioUnitario,
			nuevoPedido.CantidadDinero,
		)
		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoInicial)

	case "2": // Nuevo pedido
		return sm.handleTipoServicio(ctx, telefono, mensaje)

	case "3": // Actualizar datos
		sm.sender.SendMessage(telefono, "Por favor, escribe tu nombre completo (Apellido Paterno, Apellido Materno, Nombre)")
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoNombre)

	default:
		sm.sender.SendMessage(telefono, "Opción no válida. Por favor elige 1, 2 o 3.")
		return nil
	}
}

func (sm *StateMachine) handleNombre(ctx context.Context, telefono, mensaje string) error {
	partes := strings.Split(mensaje, " ")
	if len(partes) < 2 {
		sm.sender.SendMessage(telefono, "Por favor, ingresa al menos un nombre y un apellido.")
		return nil
	}

	cliente := sm.session.ClienteActual
	cliente.Nombre = partes[len(partes)-1] // El último elemento es el nombre
	cliente.ApellidoPaterno = partes[0]
	if len(partes) > 2 {
		cliente.ApellidoMaterno = strings.Join(partes[1:len(partes)-1], " ")
	}

	if err := sm.store.ActualizarCliente(ctx, cliente); err != nil {
		return fmt.Errorf("error actualizando nombre del cliente: %w", err)
	}

	sm.sender.SendMessage(telefono, "¡Gracias! Tus datos han sido guardados.")
	return sm.handleInicial(ctx, telefono) // Volver al menú principal
}

func (sm *StateMachine) handleTipoServicio(ctx context.Context, telefono, mensaje string) error {
	// Primero, enviamos la pregunta si aún no se ha hecho.
	if sm.session.ClienteActual.EstadoConversacion != EstadoEsperandoTipo {
		msg := "Entendido. ¿Tu nuevo pedido será para:\n\n1. Tanque Estacionario\n2. Cilindro"
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoTipo)
	}

	// Una vez que el usuario responde, procesamos la opción.
	opcion := strings.TrimSpace(mensaje)
	switch opcion {
	case "1":
		sm.session.PedidoEnCurso = &store.Pedido{
			ClienteID:    sm.session.ClienteActual.ID,
			TipoServicio: "estacionario",
		}
		// El siguiente paso es preguntar cómo desea medir el pedido.
		return sm.handleEstacionarioMenu(ctx, telefono, "")
	case "2":
		sm.session.PedidoEnCurso = &store.Pedido{
			ClienteID:    sm.session.ClienteActual.ID,
			TipoServicio: "cilindro",
		}
		// El siguiente paso es preguntar si es recarga o canje.
		return sm.handleCilindroOpcion(ctx, telefono, "")
	default:
		sm.sender.SendMessage(telefono, "Opción no válida. Por favor, responde 1 para Estacionario o 2 para Cilindro.")
		return nil // No cambiamos de estado.
	}
}

func (sm *StateMachine) handleEstacionarioMenu(ctx context.Context, telefono, mensaje string) error {
	// Si el estado no es el de esperar menú, es que venimos de seleccionar "Estacionario"
	// y hay que hacer la pregunta.
	if sm.session.ClienteActual.EstadoConversacion != EstadoEstacionarioMenu {
		msg := "¿Cómo te gustaría medir tu pedido?\n\n1. Por cantidad de litros.\n2. Por cantidad de dinero.\n3. Usar el tabulador de llenado."
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioMenu)
	}

	// Si ya estamos en el estado, procesamos la respuesta.
	switch mensaje {
	case "1":
		sm.sender.SendMessage(telefono, "Por favor, indica cuántos litros deseas cargar.")
		return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioLts)
	case "2":
		sm.sender.SendMessage(telefono, "Por favor, indica el monto en dinero que deseas cargar.")
		return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioDinero)
	case "3":
		sm.sender.SendMessage(telefono, "Por favor, indica la capacidad total de tu tanque en litros (ej. 300).")
		return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioTabuladorCapacidad)
	default:
		sm.sender.SendMessage(telefono, "Opción no válida. Por favor, elige 1, 2 o 3.")
		return nil
	}
}

func (sm *StateMachine) handleEstacionarioLitros(ctx context.Context, telefono, mensaje string) error {
	litros, err := strconv.ParseFloat(mensaje, 64)
	if err != nil {
		sm.sender.SendMessage(telefono, "Por favor, ingresa una cantidad válida en litros (ej. 150.5).")
		return nil
	}

	precioActualLitro := 12.50 // Simulación
	total := litros * precioActualLitro
	sm.session.PedidoEnCurso.CantidadLitros = litros
	sm.session.PedidoEnCurso.PrecioUnitario = precioActualLitro
	sm.session.PedidoEnCurso.CantidadDinero = total

	msg := fmt.Sprintf("Confirmación de pedido:\n- %.2f litros\n- Total: $%.2f\n\n¿Es correcto?\n1. Sí\n2. No", litros, total)
	sm.sender.SendMessage(telefono, msg)
	return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioConfirmacion)
}

func (sm *StateMachine) handleEstacionarioDinero(ctx context.Context, telefono, mensaje string) error {
	dinero, err := strconv.ParseFloat(mensaje, 64)
	if err != nil {
		sm.sender.SendMessage(telefono, "Por favor, ingresa una cantidad válida en dinero (ej. 500).")
		return nil
	}

	precioActualLitro := 12.50 // Simulación
	litros := dinero / precioActualLitro
	sm.session.PedidoEnCurso.CantidadDinero = dinero
	sm.session.PedidoEnCurso.PrecioUnitario = precioActualLitro
	sm.session.PedidoEnCurso.CantidadLitros = litros

	msg := fmt.Sprintf("Confirmación de pedido:\n- $%.2f\n- Total de litros: %.2f\n\n¿Es correcto?\n1. Sí\n2. No", dinero, litros)
	sm.sender.SendMessage(telefono, msg)
	return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioConfirmacion)
}

func (sm *StateMachine) handleTabuladorCapacidad(ctx context.Context, telefono, mensaje string) error {
	capacidad, err := strconv.ParseFloat(mensaje, 64)
	if err != nil {
		return sm.sender.SendMessage(telefono, "Por favor ingresa solo números (ejemplo: 300)")
	}

	sm.session.DatosTemp["capacidad_total"] = capacidad
	msg := fmt.Sprintf(
		"¿Qué porcentaje de llenado deseas?\n"+
			"(recomendado: 85%%)\n\n"+
			"Ingresa un número entre 1 y 100")

	if err := sm.sender.SendMessage(telefono, msg); err != nil {
		return err
	}
	return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioTabuladorPorcentaje)
}

func (sm *StateMachine) handleEstacionarioConfirmacion(ctx context.Context, telefono, mensaje string) error {
	switch mensaje {
	case "1":
		// Pedido confirmado, pasar al pago
		return sm.handlePago(ctx, telefono, "")
	case "2":
		// Pedido cancelado, volver al menú de estacionario
		sm.sender.SendMessage(telefono, "Pedido cancelado. Volviendo al menú de tanque estacionario.")
		return sm.handleEstacionarioMenu(ctx, telefono, "")
	default:
		sm.sender.SendMessage(telefono, "Opción no válida. Por favor, responde 1 para Sí o 2 para No.")
		return nil
	}
}

func (sm *StateMachine) handleTabuladorPorcentaje(ctx context.Context, telefono, mensaje string) error {
	porcentaje, err := strconv.ParseFloat(mensaje, 64)
	if err != nil {
		return sm.sender.SendMessage(telefono, "Por favor ingresa solo números (ejemplo: 85)")
	}

	if porcentaje <= 0 || porcentaje > 100 {
		return sm.sender.SendMessage(telefono, "El porcentaje debe estar entre 1 y 100")
	}

	capacidadTotal := sm.session.DatosTemp["capacidad_total"].(float64)
	litrosDeseados := capacidadTotal * (porcentaje / 100)
	precioLitro := 12.50 // TODO: obtener de DB
	total := litrosDeseados * precioLitro

	msg := fmt.Sprintf(
		"📊 *Resumen del Cálculo*\n\n"+
			"• Capacidad Total: %.0f Lts\n"+
			"• Porcentaje Deseado: %.0f%%\n"+
			"• Litros a Cargar: %.0f Lts\n"+
			"• Precio por Litro: $%.2f\n"+
			"• *Total a Pagar: $%.2f*\n\n"+
			"¿Confirmas el pedido?\n"+
			"1. Sí\n"+
			"2. No",
		capacidadTotal, porcentaje, litrosDeseados, precioLitro, total)

	sm.session.PedidoEnCurso = &store.Pedido{
		ClienteID:      sm.session.ClienteActual.ID,
		TipoServicio:   "estacionario",
		CantidadLitros: litrosDeseados,
		PrecioUnitario: precioLitro,
	}

	if err := sm.sender.SendMessage(telefono, msg); err != nil {
		return err
	}
	return sm.actualizarEstado(ctx, telefono, EstadoEstacionarioConfirmacion)
}

func (sm *StateMachine) handleCilindroOpcion(ctx context.Context, telefono, mensaje string) error {
	// Primero, enviamos la pregunta si aún no se ha hecho.
	if sm.session.ClienteActual.EstadoConversacion != EstadoCilindroOpcion {
		msg := "¿Tu pedido de cilindro será para:\n\n1. Recarga (con sistema QR)\n2. Canje (cambio de cilindro)"
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoCilindroOpcion)
	}

	// Una vez que el usuario responde, procesamos la opción.
	opcion := strings.TrimSpace(mensaje)
	switch opcion {
	case "1": // Recarga
		sm.session.PedidoEnCurso.TipoServicio = "cilindro_recarga"
		return sm.handleCilindroCantidad(ctx, telefono, "") // Pasar a pedir cantidad
	case "2": // Canje
		sm.session.PedidoEnCurso.TipoServicio = "cilindro_canje"
		return sm.handleCilindroCantidad(ctx, telefono, "") // Pasar a pedir cantidad
	default:
		sm.sender.SendMessage(telefono, "Opción no válida. Por favor, responde 1 para Recarga o 2 para Canje.")
		return nil // No cambiamos de estado.
	}
}

func (sm *StateMachine) handleCilindroCantidad(ctx context.Context, telefono, mensaje string) error {
	// Si el estado no es el de esperar cantidad, es que venimos de seleccionar
	// el tipo de servicio de cilindro y hay que hacer la pregunta.
	if sm.session.ClienteActual.EstadoConversacion != EstadoCilindroCantidad {
		msg := "¿Cuántos cilindros deseas pedir?"
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoCilindroCantidad)
	}

	// Si ya estamos en el estado, procesamos la respuesta.
	cantidad, err := strconv.Atoi(mensaje)
	if err != nil {
		sm.sender.SendMessage(telefono, "Por favor, ingresa un número válido de cilindros (ej. 2).")
		return nil
	}

	if cantidad <= 0 {
		sm.sender.SendMessage(telefono, "La cantidad debe ser de al menos 1 cilindro.")
		return nil
	}

	sm.session.PedidoEnCurso.CantidadCilindros = cantidad

	// Siguiente paso: el pago.
	return sm.handlePago(ctx, telefono, "")
}

func (sm *StateMachine) handleConfirmacionQR(ctx context.Context, telefono, mensaje string) error {
	switch strings.ToUpper(mensaje) {
	case "1", "SI", "SÍ":
		return sm.handlePago(ctx, telefono, "1") // Default a efectivo
	case "2", "NO":
		return sm.handleTipoServicio(ctx, telefono, "CILINDRO")
	default:
		return sm.sender.SendMessage(telefono,
			"Por favor responde:\n1. Sí\n2. No")
	}
}

func (sm *StateMachine) handleFotoSello(ctx context.Context, telefono, mensaje string) error {
	// TODO: Procesar foto cuando esté disponible
	msg := "¡Gracias! Tu reporte ha sido actualizado con la foto.\n" +
		"Un supervisor se comunicará contigo pronto para resolver el caso."

	if err := sm.sender.SendMessage(telefono, msg); err != nil {
		return err
	}
	return sm.actualizarEstado(ctx, telefono, EstadoInicial)
}

func (sm *StateMachine) handleConfirmacionEntrega(ctx context.Context, telefono, mensaje string) error {
	switch strings.ToUpper(mensaje) {
	case "1", "SI", "SÍ":
		sm.session.PedidoEnCurso.Estado = "entregado"
		if err := sm.store.ActualizarPedido(ctx, sm.session.PedidoEnCurso); err != nil {
			return err
		}

		msg := "¡Gracias por confirmar la entrega!\n" +
			"¿Deseas calificar nuestro servicio?\n" +
			"1. ⭐⭐⭐⭐⭐ Excelente\n" +
			"2. ⭐⭐⭐⭐ Muy bueno\n" +
			"3. ⭐⭐⭐ Regular\n" +
			"4. ⭐⭐ Malo\n" +
			"5. ⭐ Muy malo"

		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoInicial)

	case "2", "NO":
		msg := "Por favor indícanos qué problema tuviste con la entrega.\n" +
			"Un supervisor revisará tu caso inmediatamente."

		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoReportandoSello)

	default:
		return sm.sender.SendMessage(telefono,
			"Por favor responde:\n1. Sí\n2. No")
	}
}

func (sm *StateMachine) handleReporteSello(ctx context.Context, telefono string) error {
	reporte := &store.ReporteSello{
		ClienteID:    sm.session.ClienteActual.ID,
		Estado:      "pendiente",
		Descripcion: "Reporte de sello violado",
		FechaReporte: time.Now(),
	}
	if err := sm.store.CrearReporteSello(ctx, reporte); err != nil {
		return err
	}

	msg := "⚠️ *Reporte Recibido*\n\n" +
		"Tu caso ha sido registrado con prioridad alta.\n" +
		"Un supervisor se comunicará contigo en breve.\n\n" +
		"¿Deseas enviar una foto del sello?\n" +
		"1. Sí\n2. No"

	if err := sm.sender.SendMessage(telefono, msg); err != nil {
		return err
	}
	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoFotoSello)
}

func (sm *StateMachine) actualizarEstado(ctx context.Context, telefono, nuevoEstado string) error {
	return sm.store.ActualizarEstadoCliente(ctx, telefono, nuevoEstado)
}

// --- Implementaciones pendientes para Fases Futuras ---

func (sm *StateMachine) handleFotoCasa(ctx context.Context, telefono, mensaje string) error {
	sm.sender.SendMessage(telefono, "Función de foto de casa pendiente.")
	return sm.actualizarEstado(ctx, telefono, EstadoInicial)
}

func (sm *StateMachine) handleConfirmacionFotoCasa(ctx context.Context, telefono, mensaje string) error {
	sm.sender.SendMessage(telefono, "Función de confirmación de foto pendiente.")
	return sm.actualizarEstado(ctx, telefono, EstadoInicial)
}

func (sm *StateMachine) handlePago(ctx context.Context, telefono, mensaje string) error {
	// Si el estado no es el de esperar pago, hacemos la pregunta.
	if sm.session.ClienteActual.EstadoConversacion != EstadoEsperandoPago {
		msg := "¿Cuál será tu método de pago?\n\n1. Tarjeta\n2. Efectivo"
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoPago)
	}

	// Si ya estamos en el estado, procesamos la respuesta.
	switch mensaje {
	case "1":
		sm.session.PedidoEnCurso.MetodoPago = "tarjeta"
	case "2":
		sm.session.PedidoEnCurso.MetodoPago = "efectivo"
	default:
		sm.sender.SendMessage(telefono, "Opción no válida. Por favor, elige 1 para Tarjeta o 2 para Efectivo.")
		return nil
	}

	// Siguiente paso: pedir la dirección.
	return sm.handleDireccion(ctx, telefono, "")
}

func (sm *StateMachine) handleDireccion(ctx context.Context, telefono, mensaje string) error {
	// Si el estado no es el de esperar dirección, hacemos la pregunta.
	if sm.session.ClienteActual.EstadoConversacion != EstadoEsperandoDireccion {
		msg := "Por favor, escribe tu dirección completa (calle, número, colonia, etc.)."
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoDireccion)
	}

	// Si ya estamos en el estado, guardamos la dirección.
	if strings.TrimSpace(mensaje) == "" {
		sm.sender.SendMessage(telefono, "La dirección no puede estar vacía. Por favor, inténtalo de nuevo.")
		return nil
	}
	sm.session.PedidoEnCurso.Direccion = mensaje

	// Siguiente paso: confirmar la dirección.
	return sm.handleConfirmacionDireccion(ctx, telefono, "")
}

func (sm *StateMachine) handleConfirmacionDireccion(ctx context.Context, telefono, mensaje string) error {
	// Si el estado no es el de esperar confirmación, hacemos la pregunta.
	if sm.session.ClienteActual.EstadoConversacion != EstadoConfirmandoDireccion {
		msg := fmt.Sprintf("Tu dirección es:\n\n*%s*\n\n¿Es correcta?\n1. Sí\n2. No, quiero cambiarla.", sm.session.PedidoEnCurso.Direccion)
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoConfirmandoDireccion)
	}

	// Si ya estamos en el estado, procesamos la respuesta.
	switch mensaje {
	case "1":
		// La dirección es correcta, procedemos a la confirmación final del pedido.
		return sm.handleConfirmacionFinal(ctx, telefono)
	case "2":
		// El usuario quiere cambiar la dirección, volvemos a preguntar.
		return sm.handleDireccion(ctx, telefono, "")
	default:
		sm.sender.SendMessage(telefono, "Opción no válida. Por favor, responde 1 para Sí o 2 para No.")
		return nil
	}
}

func (sm *StateMachine) handleColorFachada(ctx context.Context, telefono, mensaje string) error {
	sm.sender.SendMessage(telefono, "Función de color de fachada pendiente.")
	return sm.actualizarEstado(ctx, telefono, EstadoInicial)
}
