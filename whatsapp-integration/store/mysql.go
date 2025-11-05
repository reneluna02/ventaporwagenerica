package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type MySQLStore struct {
	db *sql.DB
}

func NewMySQLStore(cfg Config) (*MySQLStore, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Database)

	// Agregar parámetros adicionales
	if len(cfg.Params) > 0 {
		dsn += "?"
		for k, v := range cfg.Params {
			dsn += fmt.Sprintf("%s=%s&", k, v)
		}
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("error conectando a MySQL: %w", err)
	}

	// Configurar pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &MySQLStore{db: db}, nil
}

func (s *MySQLStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *MySQLStore) Close() error {
	return s.db.Close()
}

func (s *MySQLStore) GetClientePorTelefono(ctx context.Context, telefono string) (*Cliente, error) {
	query := `
		SELECT id, numero_telefono, nombre, apellido_paterno, apellido_materno, 
			   estado_conversacion, created_at, updated_at
		FROM clientes 
		WHERE numero_telefono = ?`

	row := s.db.QueryRowContext(ctx, query, telefono)

	cliente := &Cliente{}
	err := row.Scan(
		&cliente.ID,
		&cliente.NumeroTelefono,
		&cliente.Nombre,
		&cliente.ApellidoPaterno,
		&cliente.ApellidoMaterno,
		&cliente.EstadoConversacion,
		&cliente.CreatedAt,
		&cliente.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil // cliente no encontrado
	}
	if err != nil {
		return nil, fmt.Errorf("error escaneando cliente: %w", err)
	}
	return cliente, nil
}

func (s *MySQLStore) CrearCliente(ctx context.Context, cliente *Cliente) error {
	query := `
		INSERT INTO clientes (
			numero_telefono, nombre, apellido_paterno, apellido_materno, estado_conversacion
		) VALUES (?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		cliente.NumeroTelefono,
		cliente.Nombre,
		cliente.ApellidoPaterno,
		cliente.ApellidoMaterno,
		cliente.EstadoConversacion,
	)
	if err != nil {
		return fmt.Errorf("error insertando cliente: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error obteniendo ID insertado: %w", err)
	}
	cliente.ID = int(id)
	return nil
}

func (s *MySQLStore) ActualizarCliente(ctx context.Context, cliente *Cliente) error {
	if cliente == nil || cliente.ID == 0 {
		return fmt.Errorf("cliente inválido para actualizar")
	}

	query := `
		UPDATE clientes
		SET nombre = ?, apellido_paterno = ?, apellido_materno = ?,
			color_puerta = ?, color_fachada = ?, codigo_rojo = ?, estado_conversacion = ?, updated_at = NOW()
		WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query,
		cliente.Nombre,
		cliente.ApellidoPaterno,
		cliente.ApellidoMaterno,
		cliente.ColorPuerta,
		cliente.ColorFachada,
		cliente.CodigoRojo,
		cliente.EstadoConversacion,
		cliente.ID,
	)
	if err != nil {
		return fmt.Errorf("error actualizando cliente: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error verificando actualización de cliente: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("cliente no encontrado: %d", cliente.ID)
	}
	return nil
}

func (s *MySQLStore) ActualizarEstadoCliente(ctx context.Context, telefono, estado string) error {
	query := `
		UPDATE clientes 
		SET estado_conversacion = ? 
		WHERE numero_telefono = ?`

	result, err := s.db.ExecContext(ctx, query, estado, telefono)
	if err != nil {
		return fmt.Errorf("error actualizando estado: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error verificando actualización: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("cliente no encontrado: %s", telefono)
	}
	return nil
}

func (s *MySQLStore) GetUltimoPedido(ctx context.Context, clienteID int) (*Pedido, error) {
	query := `
		SELECT id, cliente_id, tipo_servicio, cantidad_litros, cantidad_dinero,
			   metodo_pago, direccion, color_fachada, estado, created_at, updated_at
		FROM pedidos 
		WHERE cliente_id = ?
		ORDER BY created_at DESC
		LIMIT 1`

	row := s.db.QueryRowContext(ctx, query, clienteID)

	pedido := &Pedido{}
	err := row.Scan(
		&pedido.ID,
		&pedido.ClienteID,
		&pedido.TipoServicio,
		&pedido.CantidadLitros,
		&pedido.CantidadDinero,
		&pedido.MetodoPago,
		&pedido.Direccion,
		&pedido.ColorFachada,
		&pedido.Estado,
		&pedido.CreatedAt,
		&pedido.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("error escaneando pedido: %w", err)
	}
	return pedido, nil
}

func (s *MySQLStore) CrearPedido(ctx context.Context, pedido *Pedido) error {
	query := `
		INSERT INTO pedidos (
			cliente_id, tipo_servicio, cantidad_litros, cantidad_dinero,
			metodo_pago, direccion, color_fachada, estado
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		pedido.ClienteID,
		pedido.TipoServicio,
		pedido.CantidadLitros,
		pedido.CantidadDinero,
		pedido.MetodoPago,
		pedido.Direccion,
		pedido.ColorFachada,
		pedido.Estado,
	)
	if err != nil {
		return fmt.Errorf("error insertando pedido: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error obteniendo ID insertado: %w", err)
	}
	pedido.ID = int(id)
	return nil
}

func (s *MySQLStore) ActualizarPedido(ctx context.Context, pedido *Pedido) error {
	query := `
		UPDATE pedidos 
		SET tipo_servicio = ?, cantidad_litros = ?, cantidad_dinero = ?,
			metodo_pago = ?, direccion = ?, color_fachada = ?, estado = ?
		WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query,
		pedido.TipoServicio,
		pedido.CantidadLitros,
		pedido.CantidadDinero,
		pedido.MetodoPago,
		pedido.Direccion,
		pedido.ColorFachada,
		pedido.Estado,
		pedido.ID,
	)
	if err != nil {
		return fmt.Errorf("error actualizando pedido: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error verificando actualización: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("pedido no encontrado: %d", pedido.ID)
	}
	return nil
}

func (s *MySQLStore) CrearReporteSello(ctx context.Context, reporte *ReporteSello) error {
	// Registrar fecha de reporte en memoria (la columna en BD puede no existir aún)
	if reporte.FechaReporte.IsZero() {
		reporte.FechaReporte = time.Now()
	}

	query := `
		INSERT INTO reportes_sello (
			cliente_id, pedido_id, descripcion, estado
		) VALUES (?, ?, ?, ?)`

	result, err := s.db.ExecContext(ctx, query,
		reporte.ClienteID,
		reporte.PedidoID,
		reporte.Descripcion,
		reporte.Estado,
	)
	if err != nil {
		return fmt.Errorf("error insertando reporte: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("error obteniendo ID insertado: %w", err)
	}
	reporte.ID = int(id)
	return nil
}
