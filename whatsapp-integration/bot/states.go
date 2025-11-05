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
		// Nuevo cliente: solicitar foto de la casa o, si no, colores.
		cliente = &store.Cliente{
			NumeroTelefono:    telefono,
			EstadoConversacion: EstadoEsperandoFotoCasa,
		}
		if err := sm.store.CrearCliente(ctx, cliente); err != nil {
			return fmt.Errorf("error creando cliente: %w", err)
		}

		sm.session.ClienteActual = cliente
		// Preguntar por foto de la casa en el primer registro
		msg := "Para ayudar al repartidor, ¿puedes enviar una foto de tu casa?\n\n1. Sí, enviaré una foto\n2. No, prefiero describir colores"
		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return fmt.Errorf("error enviando pregunta de foto: %w", err)
		}
		// Actualizar estado del cliente y terminar (esperamos la respuesta)
		if err := sm.actualizarEstado(ctx, telefono, EstadoEsperandoFotoCasa); err != nil {
			return fmt.Errorf("error actualizando estado cliente: %w", err)
		}
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
		msg = "¡Bienvenido! Por favor elige:\n\n1. Nuevo pedido\n2. Registrar mis datos"
	}

	if err := sm.sender.SendMessage(telefono, msg); err != nil {
		return err
	}

	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoOpcion)
}

func (sm *StateMachine) handleOpcionInicial(ctx context.Context, telefono, mensaje string) error {
	switch mensaje {
	case "1":
		pedido, err := sm.store.GetUltimoPedido(ctx, sm.session.ClienteActual.ID)
		if err != nil {
			return err
		}
		if pedido == nil {
			return sm.handleTipoServicio(ctx, telefono, mensaje)
		}
		// Crear nuevo pedido basado en el anterior
		nuevoPedido := *pedido
		nuevoPedido.ID = 0 // nueva entrada
		nuevoPedido.Estado = "pendiente"
		if err := sm.store.CrearPedido(ctx, &nuevoPedido); err != nil {
			return err
		}
		sm.sender.SendMessage(telefono, "¡Pedido confirmado! En breve recibirás confirmación de entrega.")
		return sm.actualizarEstado(ctx, telefono, EstadoInicial)

	case "2":
		return sm.handleTipoServicio(ctx, telefono, mensaje)

	case "3":
		sm.sender.SendMessage(telefono, "Por favor, escribe tu nombre completo (Apellido Paterno, Apellido Materno, Nombre)")
		return sm.actualizarEstado(ctx, telefono, EstadoEsperandoNombre)

	default:
		sm.sender.SendMessage(telefono, "Opción no válida. Por favor elige 1, 2 o 3.")
		return nil
	}
}

func (sm *StateMachine) handleTipoServicio(ctx context.Context, telefono, mensaje string) error {
	msg := "¿Tu servicio será para Tanque Estacionario o Cilindro?"
	if err := sm.sender.SendMessage(telefono, msg); err != nil {
		return err
	}
	return sm.actualizarEstado(ctx, telefono, EstadoEsperandoTipo)
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
	switch mensaje {
	case "1": // Recarga
		msg := "📱 *Sistema de Recarga con QR*\n\n" +
			"Te asignaremos un código QR único para rastrear tu cilindro.\n\n" +
			"¿Cuántos cilindros deseas recargar?\n" +
			"(máximo 3 por servicio)"

		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoCilindroCantidad)

	case "2": // Canje
		msg := "Has elegido canje de cilindro.\n\n" +
			"¿Cuántos cilindros deseas canjear?\n" +
			"(máximo 3 por servicio)"

		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoCilindroCantidad)

	default:
		return sm.sender.SendMessage(telefono, 
			"Por favor selecciona:\n1. Recarga\n2. Canje")
	}
}

func (sm *StateMachine) handleCilindroCantidad(ctx context.Context, telefono, mensaje string) error {
	cantidad, err := strconv.Atoi(mensaje)
	if err != nil {
		return sm.sender.SendMessage(telefono, 
			"Por favor ingresa solo números (ejemplo: 2)")
	}

	if cantidad < 1 || cantidad > 3 {
		return sm.sender.SendMessage(telefono,
			"La cantidad debe ser entre 1 y 3 cilindros")
	}

	sm.session.PedidoEnCurso.CantidadCilindros = cantidad

	if sm.session.PedidoEnCurso.TipoServicio == "cilindro_recarga" {
		// Generar QRs
		codigos := make([]string, cantidad)
		for i := 0; i < cantidad; i++ {
			codigos[i] = fmt.Sprintf("CIL-%d-%d", time.Now().Unix(), i+1)
		}
		sm.session.PedidoEnCurso.CodigosQR = codigos

		msg := fmt.Sprintf(
			"Se han generado %d códigos QR para tus cilindros.\n"+
				"Por favor guarda estas imágenes.\n\n"+
				"¿Deseas continuar?\n"+
				"1. Sí\n"+
				"2. No",
			cantidad)

		if err := sm.sender.SendMessage(telefono, msg); err != nil {
			return err
		}
		return sm.actualizarEstado(ctx, telefono, EstadoCilindroConfirmacionQR)
	}

	// Si es canje, ir directo a pago
	return sm.handlePago(ctx, telefono, "1") // Default a efectivo
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
		PedidoID:    sm.session.PedidoEnCurso.ID,
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
