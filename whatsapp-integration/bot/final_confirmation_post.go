
package bot

import (
	"context"
	"fmt"
)

func (sm *StateMachine) handleConfirmacionFinalPost(ctx context.Context, telefono string, mensaje string) error {
	switch mensaje {
	case "1": // Sí, confirmar
		pedido := sm.session.PedidoEnCurso
		// Asignar estado inicial según el tipo de servicio.
		if pedido.TipoServicio == "cilindro_recarga" {
			pedido.Estado = "pendiente_recoleccion"
		} else {
			pedido.Estado = "pendiente"
		}

		if err := sm.store.CrearPedido(ctx, pedido); err != nil {
			return fmt.Errorf("error al guardar el pedido en la base de datos: %w", err)
		}
		msg := "¡Tu pedido ha sido confirmado! En breve recibirás una notificación sobre la entrega."
		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoInicial)
	case "2": // No, cancelar
		msg := "Tu pedido ha sido cancelado. Puedes iniciar uno nuevo cuando quieras."
		sm.sender.SendMessage(telefono, msg)
		return sm.actualizarEstado(ctx, telefono, EstadoInicial)
	default:
		sm.sender.SendMessage(telefono, "Opción no válida. Por favor, responde 1 para confirmar o 2 para cancelar.")
		return nil
	}
}
