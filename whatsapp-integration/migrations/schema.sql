-- Esquema de base de datos para el bot de WhatsApp
-- Ejecutar en orden para configurar la base de datos

-- Tabla de clientes
CREATE TABLE IF NOT EXISTS clientes (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    numero_telefono VARCHAR(20) NOT NULL UNIQUE,
    nombre VARCHAR(100),
    apellido_paterno VARCHAR(100),
    apellido_materno VARCHAR(100),
    estado_conversacion VARCHAR(50) NOT NULL DEFAULT 'INICIO',
    ultima_direccion TEXT,
    color_puerta VARCHAR(50),
    color_fachada VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE INDEX idx_telefono (numero_telefono)
);

-- Tabla de precios históricos
CREATE TABLE IF NOT EXISTS precios (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    tipo ENUM('litro', 'cilindro_10kg', 'cilindro_20kg', 'cilindro_30kg') NOT NULL,
    precio DECIMAL(10,2) NOT NULL,
    fecha_inicio TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    fecha_fin TIMESTAMP NULL,
    activo BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_activo (activo)
);

-- Tabla de pedidos
CREATE TABLE IF NOT EXISTS pedidos (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    cliente_id INTEGER NOT NULL,
    tipo_servicio ENUM('estacionario', 'cilindro') NOT NULL,
    subtipo_servicio ENUM('recarga', 'canje') NULL,
    cantidad_litros DECIMAL(10,2),
    cantidad_dinero DECIMAL(10,2),
    precio_unitario DECIMAL(10,2) NOT NULL,
    metodo_pago ENUM('efectivo', 'tarjeta'),
    direccion TEXT,
    lat_long VARCHAR(50),
    maps_url TEXT,
    street_view_url TEXT,
    color_puerta VARCHAR(50),
    color_fachada VARCHAR(50),
    estado ENUM(
        'pendiente',
        'recoleccion_programada',
        'tanque_recogido',
        'en_planta',
        'en_ruta_entrega',
        'entregado',
        'cancelado'
    ) NOT NULL DEFAULT 'pendiente',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (cliente_id) REFERENCES clientes(id),
    INDEX idx_cliente (cliente_id),
    INDEX idx_estado (estado)
);

-- Tabla de tanques y códigos QR
CREATE TABLE IF NOT EXISTS tanques (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    pedido_id INTEGER NOT NULL,
    codigo_qr VARCHAR(100) UNIQUE NOT NULL,
    capacidad DECIMAL(10,2),
    tipo ENUM('estacionario', 'cilindro_10kg', 'cilindro_20kg', 'cilindro_30kg') NOT NULL,
    estado ENUM(
        'con_cliente',
        'recolectado',
        'en_planta',
        'en_ruta',
        'entregado'
    ) NOT NULL DEFAULT 'con_cliente',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (pedido_id) REFERENCES pedidos(id),
    INDEX idx_qr (codigo_qr),
    INDEX idx_estado (estado)
);

-- Tabla de reportes de sello
CREATE TABLE IF NOT EXISTS reportes_sello (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    cliente_id INTEGER NOT NULL,
    pedido_id INTEGER NOT NULL,
    tanque_id INTEGER NOT NULL,
    tipo_reporte ENUM('sello_violado', 'tanque_danado', 'otro') NOT NULL,
    descripcion TEXT,
    foto_url TEXT,
    estado ENUM('pendiente', 'en_revision', 'resuelto', 'cancelado') NOT NULL DEFAULT 'pendiente',
    resolucion TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (cliente_id) REFERENCES clientes(id),
    FOREIGN KEY (pedido_id) REFERENCES pedidos(id),
    FOREIGN KEY (tanque_id) REFERENCES tanques(id),
    INDEX idx_estado (estado)
);

-- Tabla de tabulador de capacidades (para cálculos de llenado)
CREATE TABLE IF NOT EXISTS tabulador_capacidades (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    capacidad_total DECIMAL(10,2) NOT NULL,
    porcentaje_recomendado DECIMAL(5,2) NOT NULL DEFAULT 85.00,
    precio_actual DECIMAL(10,2) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Datos iniciales
INSERT INTO precios (tipo, precio, activo) VALUES
('litro', 12.50, TRUE),
('cilindro_10kg', 250.00, TRUE),
('cilindro_20kg', 480.00, TRUE),
('cilindro_30kg', 720.00, TRUE);

INSERT INTO tabulador_capacidades (capacidad_total, porcentaje_recomendado, precio_actual) VALUES
(300, 85.00, 12.50),
(500, 85.00, 12.50),
(1000, 85.00, 12.50);