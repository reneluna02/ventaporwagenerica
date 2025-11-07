package migrations

import (
	"database/sql"
	"fmt"
)

const (
	createClientesTable = `
	CREATE TABLE IF NOT EXISTS clientes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		numero_telefono TEXT NOT NULL UNIQUE,
		nombre TEXT,
		apellido_paterno TEXT,
		apellido_materno TEXT,
		estado_conversacion TEXT,
		color_puerta TEXT,
		color_fachada TEXT,
		codigo_rojo BOOLEAN DEFAULT FALSE,
		strikes INTEGER DEFAULT 0,
		bloqueado BOOLEAN DEFAULT FALSE,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);`

	createPedidosTable = `
	CREATE TABLE IF NOT EXISTS pedidos (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cliente_id INTEGER,
		tipo_servicio TEXT,
		cantidad_litros REAL,
		cantidad_dinero REAL,
		precio_unitario REAL,
		metodo_pago TEXT,
		direccion TEXT,
		color_fachada TEXT,
		color_puerta TEXT,
		codigo_rojo BOOLEAN,
		cantidad_cilindros INTEGER,
		codigos_qr TEXT,
		estado TEXT,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(cliente_id) REFERENCES clientes(id)
	);`

	createReportesSelloTable = `
	CREATE TABLE IF NOT EXISTS reportes_sello (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		cliente_id INTEGER,
		pedido_id INTEGER,
		descripcion TEXT,
		estado TEXT,
		fecha_reporte TIMESTAMP,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(cliente_id) REFERENCES clientes(id),
		FOREIGN KEY(pedido_id) REFERENCES pedidos(id)
	);`
)

// RunSQLiteMigrations ejecuta las migraciones para una base de datos SQLite
func RunSQLiteMigrations(db *sql.DB) error {
	tables := []string{
		createClientesTable,
		createPedidosTable,
		createReportesSelloTable,
	}

	for _, table := range tables {
		if _, err := db.Exec(table); err != nil {
			return fmt.Errorf("error creando tabla: %w", err)
		}
	}

	fmt.Println("Migraciones de SQLite completadas exitosamente.")
	return nil
}
