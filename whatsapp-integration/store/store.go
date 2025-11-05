package store

import (
	"context"
	"fmt"
	"time"
)

// Cliente representa un cliente en la base de datos
type Cliente struct {
	ID                 int
	NumeroTelefono     string
	Nombre             string
	ApellidoPaterno    string
	ApellidoMaterno    string
	EstadoConversacion string
	ColorPuerta        string
	ColorFachada       string
	CodigoRojo         bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// Pedido representa un pedido en la base de datos
type Pedido struct {
	ID                int
	ClienteID         int
	TipoServicio      string // "estacionario" o "cilindro"
	CantidadLitros    float64
	CantidadDinero    float64
	PrecioUnitario    float64
	MetodoPago        string
	Direccion         string
	ColorFachada      string
	ColorPuerta       string
	CodigoRojo        bool
	CantidadCilindros int
	CodigosQR         string // comma-separated codes; puede evolucionar a JSON
	Estado            string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// ReporteSello representa un reporte de sello violado
type ReporteSello struct {
	ID           int
	ClienteID    int
	PedidoID     *int // opcional
	Descripcion  string
	Estado       string
	FechaReporte time.Time
	CreatedAt    time.Time
}

// Store define la interfaz para acceder a la base de datos
type Store interface {
	// Métodos para Cliente
	GetClientePorTelefono(ctx context.Context, telefono string) (*Cliente, error)
	CrearCliente(ctx context.Context, cliente *Cliente) error
	ActualizarCliente(ctx context.Context, cliente *Cliente) error
	ActualizarEstadoCliente(ctx context.Context, telefono, estado string) error

	// Métodos para Pedido
	GetUltimoPedido(ctx context.Context, clienteID int) (*Pedido, error)
	CrearPedido(ctx context.Context, pedido *Pedido) error
	ActualizarPedido(ctx context.Context, pedido *Pedido) error

	// Métodos para ReporteSello
	CrearReporteSello(ctx context.Context, reporte *ReporteSello) error

	// Utilidades
	Ping(ctx context.Context) error
	Close() error
}

// Config contiene la configuración para conectar a la base de datos
type Config struct {
	Driver   string // "mysql" o "sqlite3"
	Host     string
	Port     string
	User     string
	Password string
	Database string
	Params   map[string]string
}

// NewStore crea una nueva instancia de Store según el driver
func NewStore(cfg Config) (Store, error) {
	switch cfg.Driver {
	case "mysql":
		return NewMySQLStore(cfg)
	case "sqlite3":
		return NewSQLiteStore(cfg)
	default:
		return nil, fmt.Errorf("driver no soportado: %s", cfg.Driver)
	}
}
