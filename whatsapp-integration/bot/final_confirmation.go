
package bot

import (
	"context"
	"fmt"
)

func (sm *StateMachine) handleConfirmacionFinal(ctx context.Context, telefono string) error {
	pedido := sm.session.PedidoEnCurso
	resumen := fmt.Sprintf(
		"📝 *Resumen de tu Pedido*\n\n"+
			"  - *Servicio:* %s\n"+
			"  - *Cantidad:* %.2f Lts\n"+
			"  - *Total a Pagar:* $%.2f\n"+
			"  - *Método de Pago:* %s\n"+
			"  - *Dirección de Entrega:* %s\n\n"+
			"*Importante:* Nuestro repartidor solo podrá esperar un máximo de 10 minutos en tu domicilio.\n\n"+
			"¿Confirmas tu pedido?\n1. Sí, confirmar\n2. No, cancelar",
		pedido.TipoServicio,
		pedido.CantidadLitros,
		pedido.CantidadDinero,
		pedido.MetodoPago,
		pedido.Direccion,
	)

	if err := sm.sender.SendMessage(telefono, resumen); err != nil {
		return err
	}

	return sm.actualizarEstado(ctx, telefono, "CONFIRMANDO_PEDIDO_FINAL")
}
