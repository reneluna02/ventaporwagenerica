package delivery

import (
	"context"
	"fmt"
	"time"
	"encoding/json"
	"net/http"
	"net/url"

	"example.com/whatsapp-integration/store"
)

type MapsService struct {
	apiKey     string
	httpClient *http.Client
	store      store.Store
}

type Location struct {
	Lat     float64 `json:"lat"`
	Lng     float64 `json:"lng"`
	Address string  `json:"address"`
}

type RouteInfo struct {
	Distance      float64 `json:"distance"`      // en kilómetros
	Duration      int     `json:"duration"`      // en minutos
	EstimatedTime time.Time `json:"estimatedTime"`
}

type DeliveryRoute struct {
	DriverID    string       `json:"driverId"`
	Stops       []RouteStop  `json:"stops"`
	TotalTime   int         `json:"totalTime"`    // en minutos
	TotalDistance float64   `json:"totalDistance"` // en kilómetros
}

type RouteStop struct {
	PedidoID     string    `json:"pedidoId"`
	Location     Location  `json:"location"`
	EstimatedTime time.Time `json:"estimatedTime"`
	Status       string    `json:"status"` // pending, arriving, delivered, cancelled
}

func NewMapsService(apiKey string, store store.Store) *MapsService {
	return &MapsService{
		apiKey: apiKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		store: store,
	}
}

// ValidateAddress verifica y normaliza una dirección
func (s *MapsService) ValidateAddress(ctx context.Context, address string) (*Location, error) {
	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s",
		url.QueryEscape(address),
		s.apiKey,
	)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error geocoding address: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			FormattedAddress string `json:"formatted_address"`
			Geometry struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("error decoding response: %w", err)
	}

	if len(result.Results) == 0 {
		return nil, fmt.Errorf("no results found for address")
	}

	return &Location{
		Lat: result.Results[0].Geometry.Location.Lat,
		Lng: result.Results[0].Geometry.Location.Lng,
		Address: result.Results[0].FormattedAddress,
	}, nil
}

// OptimizeRoute genera una ruta óptima para un conjunto de pedidos
func (s *MapsService) OptimizeRoute(ctx context.Context, pedidos []*store.Pedido) (*DeliveryRoute, error) {
	if len(pedidos) == 0 {
		return nil, fmt.Errorf("no orders to route")
	}

	// Obtener coordenadas de cada pedido
	stops := make([]RouteStop, len(pedidos))
	for i, pedido := range pedidos {
		loc, err := s.ValidateAddress(ctx, pedido.Direccion)
		if err != nil {
			return nil, fmt.Errorf("error validating address for order %s: %w", pedido.ID, err)
		}

		stops[i] = RouteStop{
			PedidoID: pedido.ID,
			Location: *loc,
			Status:   "pending",
		}
	}

	// Llamar a la API de Directions para optimizar la ruta
	optimizedStops, err := s.getOptimizedRoute(stops)
	if err != nil {
		return nil, fmt.Errorf("error optimizing route: %w", err)
	}

	// Calcular tiempos estimados
	now := time.Now()
	currentTime := now
	for i := range optimizedStops {
		if i > 0 {
			// Calcular tiempo entre paradas
			duration, err := s.getRouteDuration(
				optimizedStops[i-1].Location,
				optimizedStops[i].Location,
			)
			if err != nil {
				return nil, err
			}
			currentTime = currentTime.Add(time.Duration(duration) * time.Minute)
		}
		optimizedStops[i].EstimatedTime = currentTime
	}

	return &DeliveryRoute{
		Stops: optimizedStops,
		TotalTime: int(currentTime.Sub(now).Minutes()),
	}, nil
}

// UpdateDeliveryStatus actualiza el estado de entrega y notifica al cliente
func (s *MapsService) UpdateDeliveryStatus(ctx context.Context, pedidoID string, status string, location Location) error {
	pedido, err := s.store.GetPedido(ctx, pedidoID)
	if err != nil {
		return fmt.Errorf("error getting order: %w", err)
	}

	// Actualizar estado
	pedido.Estado = status
	if err := s.store.ActualizarPedido(ctx, pedido); err != nil {
		return fmt.Errorf("error updating order: %w", err)
	}

	// Obtener tiempo estimado
	eta, err := s.getRouteDuration(location, Location{
		Lat: pedido.Lat,
		Lng: pedido.Lng,
		Address: pedido.Direccion,
	})
	if err != nil {
		return fmt.Errorf("error calculating ETA: %w", err)
	}

	// Enviar notificación según el estado
	var msg string
	switch status {
	case "en_ruta":
		msg = fmt.Sprintf(
			"🚛 *Tu pedido está en camino*\n\n"+
				"Tiempo estimado de llegada: %d minutos\n"+
				"Te avisaremos cuando estemos cerca.",
			eta)

	case "llegando":
		msg = "🏃 *¡Prepárate!*\n\n" +
			"El repartidor está a menos de 5 minutos.\n" +
			"Por favor ten el pago listo."

	case "esperando":
		msg = "🔔 *¡Hemos llegado!*\n\n" +
			"El repartidor está esperando.\n" +
			"Tienes 5 minutos para confirmar la recepción\n" +
			"o el pedido será cancelado."

	case "entregado":
		msg = "✅ *Entrega Confirmada*\n\n" +
			"¡Gracias por tu preferencia!\n" +
			"¿Deseas calificar nuestro servicio?"

	case "cancelado":
		msg = "❌ *Pedido Cancelado*\n\n" +
			"No se recibió confirmación en el tiempo establecido.\n" +
			"Por favor contacta a soporte si esto es un error."
	}

	// TODO: Enviar mensaje vía WhatsApp
	return nil
}

// AutoCancelUnconfirmed cancela pedidos no confirmados después de 5 minutos
func (s *MapsService) AutoCancelUnconfirmed(ctx context.Context) error {
	pedidos, err := s.store.GetPedidosEstado(ctx, "esperando")
	if err != nil {
		return fmt.Errorf("error getting waiting orders: %w", err)
	}

	now := time.Now()
	for _, pedido := range pedidos {
		if now.Sub(pedido.UltimaActualizacion) > 5*time.Minute {
			if err := s.UpdateDeliveryStatus(ctx, pedido.ID, "cancelado", Location{}); err != nil {
				// Log error pero continuar con otros pedidos
				fmt.Printf("Error canceling order %s: %v\n", pedido.ID, err)
			}
		}
	}

	return nil
}

// getOptimizedRoute llama a la API de Directions para optimizar la ruta
func (s *MapsService) getOptimizedRoute(stops []RouteStop) ([]RouteStop, error) {
	// TODO: Implementar llamada real a la API
	// Por ahora retorna las paradas en el mismo orden
	return stops, nil
}

// getRouteDuration calcula el tiempo entre dos ubicaciones
func (s *MapsService) getRouteDuration(origin, dest Location) (int, error) {
	url := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/directions/json?origin=%f,%f&destination=%f,%f&key=%s",
		origin.Lat, origin.Lng,
		dest.Lat, dest.Lng,
		s.apiKey,
	)

	resp, err := s.httpClient.Get(url)
	if err != nil {
		return 0, fmt.Errorf("error getting directions: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Routes []struct {
			Legs []struct {
				Duration struct {
					Value int `json:"value"` // en segundos
				} `json:"duration"`
			} `json:"legs"`
		} `json:"routes"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("error decoding response: %w", err)
	}

	if len(result.Routes) == 0 || len(result.Routes[0].Legs) == 0 {
		return 0, fmt.Errorf("no route found")
	}

	// Convertir segundos a minutos
	return result.Routes[0].Legs[0].Duration.Value / 60, nil
}