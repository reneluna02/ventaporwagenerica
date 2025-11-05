package delivery

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"example.com/whatsapp-integration/store"
)

type POSService struct {
	endpoint   string
	apiKey     string
	httpClient *http.Client
	store      store.Store
	maps       *MapsService

	// Cache de rutas activas
	activeRoutes map[string]*DeliveryRoute
	routesMutex  sync.RWMutex
}

type POSConfig struct {
	MaxPedidosPorRuta   int     `json:"maxPedidosPorRuta"`
	TiempoMaximoRuta    int     `json:"tiempoMaximoRuta"`    // en minutos
	DistanciaMaximaRuta float64 `json:"distanciaMaximaRuta"` // en kilómetros
	IntervaloSync       int     `json:"intervaloSync"`       // en segundos
}

func NewPOSService(endpoint, apiKey string, store store.Store, maps *MapsService) *POSService {
	return &POSService{
		endpoint: endpoint,
		apiKey:   apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		store:        store,
		maps:         maps,
		activeRoutes: make(map[string]*DeliveryRoute),
	}
}

// SyncWithPOS sincroniza pedidos con la terminal punto de venta
func (s *POSService) SyncWithPOS(ctx context.Context) error {
	// Obtener configuración actual
	config, err := s.getPOSConfig()
	if err != nil {
		return fmt.Errorf("error getting POS config: %w", err)
	}

	// Obtener pedidos pendientes
	pedidos, err := s.store.GetPedidosEstado(ctx, "pendiente")
	if err != nil {
		return fmt.Errorf("error getting pending orders: %w", err)
	}

	// Agrupar pedidos por zona/ruta
	grupos := s.agruparPedidos(pedidos, config)

	// Para cada grupo, optimizar ruta y asignar tiempos
	for _, grupo := range grupos {
		ruta, err := s.maps.OptimizeRoute(ctx, grupo)
		if err != nil {
			return fmt.Errorf("error optimizing route: %w", err)
		}

		// Guardar ruta activa
		s.routesMutex.Lock()
		s.activeRoutes[ruta.DriverID] = ruta
		s.routesMutex.Unlock()

		// Notificar a los clientes
		for i, stop := range ruta.Stops {
			pedido, err := s.store.GetPedido(ctx, stop.PedidoID)
			if err != nil {
				continue
			}

			var msg string
			eta := int(stop.EstimatedTime.Sub(time.Now()).Minutes())

			if i == len(ruta.Stops)-1 {
				// Último pedido de la ruta
				msg = fmt.Sprintf(
					"🚛 *Actualización de tu pedido*\n\n"+
						"¡Buenas noticias! Serás la última entrega de la ruta.\n"+
						"Tiempo estimado de llegada: %d minutos\n\n"+
						"Te mantendremos informado del progreso.",
					eta)
			} else {
				msg = fmt.Sprintf(
					"🚛 *Actualización de tu pedido*\n\n"+
						"Tu pedido ha sido asignado a una ruta de entrega.\n"+
						"Tiempo estimado de llegada: %d minutos\n\n"+
						"Te avisaremos cuando estemos cerca.",
					eta)
			}

			// TODO: Enviar mensaje vía WhatsApp
		}
	}

	return nil
}

// StartDeliveryTracking inicia el monitoreo de entregas
func (s *POSService) StartDeliveryTracking(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.routesMutex.RLock()
				for _, ruta := range s.activeRoutes {
					for _, stop := range ruta.Stops {
						if stop.Status != "delivered" && stop.Status != "cancelled" {
							// Verificar tiempo estimado
							if time.Now().After(stop.EstimatedTime) {
								pedido, err := s.store.GetPedido(ctx, stop.PedidoID)
								if err != nil {
									continue
								}

								if time.Now().Sub(stop.EstimatedTime) <= 5*time.Minute {
									// Menos de 5 minutos para llegar
									s.maps.UpdateDeliveryStatus(ctx, stop.PedidoID, "llegando", stop.Location)
								} else if time.Now().Sub(stop.EstimatedTime) > 5*time.Minute {
									// Más de 5 minutos esperando
									if pedido.Estado == "esperando" {
										s.maps.AutoCancelUnconfirmed(ctx)
									}
								}
							}
						}
					}
				}
				s.routesMutex.RUnlock()
			}
		}
	}()
}

// getPOSConfig obtiene la configuración de la terminal punto de venta
func (s *POSService) getPOSConfig() (*POSConfig, error) {
	url := fmt.Sprintf("%s/config?apiKey=%s", s.endpoint, s.apiKey)
	
	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error getting POS config: %w", err)
	}
	defer resp.Body.Close()

	var config POSConfig
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, fmt.Errorf("error decoding config: %w", err)
	}

	return &config, nil
}

// agruparPedidos agrupa pedidos por zona/proximidad
func (s *POSService) agruparPedidos(pedidos []*store.Pedido, config *POSConfig) [][]*store.Pedido {
	// TODO: Implementar agrupación inteligente por zonas
	// Por ahora solo divide en grupos del tamaño máximo
	
	grupos := make([][]*store.Pedido, 0)
	grupoActual := make([]*store.Pedido, 0)

	for _, pedido := range pedidos {
		grupoActual = append(grupoActual, pedido)
		
		if len(grupoActual) >= config.MaxPedidosPorRuta {
			grupos = append(grupos, grupoActual)
			grupoActual = make([]*store.Pedido, 0)
		}
	}

	if len(grupoActual) > 0 {
		grupos = append(grupos, grupoActual)
	}

	return grupos
}