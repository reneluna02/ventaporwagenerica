package store

import (
	"context"
	"database/sql"
	"fmt"
	_ "github.com/denisenkom/go-mssqldb"
)

type SQLServerStore struct {
	db *sql.DB
}

func NewSQLServerStore(cfg Config) (*SQLServerStore, error) {
	// La lógica de conexión para SQL Server iría aquí.
	return nil, fmt.Errorf("no implementado")
}

func (s *SQLServerStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *SQLServerStore) Close() error {
	return s.db.Close()
}

// --- Métodos de Cliente (pendientes de implementación) ---

func (s *SQLServerStore) GetClientePorTelefono(ctx context.Context, telefono string) (*Cliente, error) {
	return nil, fmt.Errorf("no implementado")
}

func (s *SQLServerStore) CrearCliente(ctx context.Context, cliente *Cliente) error {
	return fmt.Errorf("no implementado")
}

func (s *SQLServerStore) ActualizarCliente(ctx context.Context, cliente *Cliente) error {
	return fmt.Errorf("no implementado")
}

func (s *SQLServerStore) ActualizarEstadoCliente(ctx context.Context, telefono, estado string) error {
	return fmt.Errorf("no implementado")
}

// --- Métodos de Pedido (pendientes de implementación) ---

func (s *SQLServerStore) GetUltimoPedido(ctx context.Context, clienteID int) (*Pedido, error) {
	return nil, fmt.Errorf("no implementado")
}

func (s *SQLServerStore) GetUltimoPedidoActivo(ctx context.Context, clienteID int) (*Pedido, error) {
	return nil, fmt.Errorf("no implementado")
}

func (s *SQLServerStore) GetPedidosPorEstado(ctx context.Context, estado string) ([]*Pedido, error) {
	return nil, fmt.Errorf("no implementado")
}

func (s *SQLServerStore) CrearPedido(ctx context.Context, pedido *Pedido) error {
	return fmt.Errorf("no implementado")
}

func (s *SQLServerStore) ActualizarPedido(ctx context.Context, pedido *Pedido) error {
	return fmt.Errorf("no implementado")
}

// --- Métodos de Reporte (pendientes de implementación) ---

func (s *SQLServerStore) CrearReporteSello(ctx context.Context, reporte *ReporteSello) error {
	return fmt.Errorf("no implementado")
}
