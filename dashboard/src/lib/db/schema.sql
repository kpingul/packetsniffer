-- Captures/Sessions table
CREATE TABLE IF NOT EXISTS captures (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    sensor_os TEXT NOT NULL,
    sensor_hostname TEXT NOT NULL,
    interface_name TEXT NOT NULL,
    local_ip TEXT NOT NULL,
    start_time TEXT NOT NULL,
    duration_seconds INTEGER NOT NULL,
    packet_count INTEGER NOT NULL,
    imported_at TEXT NOT NULL DEFAULT (datetime('now')),
    filename TEXT NOT NULL
);

-- Devices table
CREATE TABLE IF NOT EXISTS devices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    capture_id INTEGER NOT NULL,
    mac TEXT NOT NULL,
    vendor TEXT,
    hostname TEXT,
    os_guess TEXT,
    os_confidence REAL,
    signals_used TEXT,
    discovery_source TEXT,
    first_seen TEXT NOT NULL,
    last_seen TEXT NOT NULL,
    FOREIGN KEY (capture_id) REFERENCES captures(id) ON DELETE CASCADE
);

-- Device IPs (many-to-one with devices)
CREATE TABLE IF NOT EXISTS device_ips (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL,
    ip TEXT NOT NULL,
    FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
);

-- Traffic protocol counts
CREATE TABLE IF NOT EXISTS protocol_counts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    capture_id INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    count INTEGER NOT NULL,
    FOREIGN KEY (capture_id) REFERENCES captures(id) ON DELETE CASCADE
);

-- Top ports
CREATE TABLE IF NOT EXISTS top_ports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    capture_id INTEGER NOT NULL,
    port INTEGER NOT NULL,
    protocol TEXT NOT NULL,
    count INTEGER NOT NULL,
    FOREIGN KEY (capture_id) REFERENCES captures(id) ON DELETE CASCADE
);

-- Top talkers
CREATE TABLE IF NOT EXISTS top_talkers (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    capture_id INTEGER NOT NULL,
    ip TEXT NOT NULL,
    bytes_sent INTEGER NOT NULL,
    bytes_received INTEGER NOT NULL,
    packets_sent INTEGER NOT NULL,
    packets_received INTEGER NOT NULL,
    FOREIGN KEY (capture_id) REFERENCES captures(id) ON DELETE CASCADE
);

-- DNS domains
CREATE TABLE IF NOT EXISTS dns_domains (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    capture_id INTEGER NOT NULL,
    domain TEXT NOT NULL,
    query_count INTEGER NOT NULL,
    querying_ips TEXT,
    FOREIGN KEY (capture_id) REFERENCES captures(id) ON DELETE CASCADE
);

-- Destinations (external IPs/domains contacted)
CREATE TABLE IF NOT EXISTS destinations (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    capture_id INTEGER NOT NULL,
    address TEXT NOT NULL,
    connection_count INTEGER NOT NULL,
    bytes_total INTEGER NOT NULL,
    FOREIGN KEY (capture_id) REFERENCES captures(id) ON DELETE CASCADE
);

-- Flows (for graph visualization)
CREATE TABLE IF NOT EXISTS flows (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    capture_id INTEGER NOT NULL,
    source_mac TEXT,
    source_ip TEXT NOT NULL,
    dest_ip TEXT NOT NULL,
    dest_port INTEGER,
    protocol TEXT,
    packet_count INTEGER NOT NULL,
    byte_count INTEGER NOT NULL,
    FOREIGN KEY (capture_id) REFERENCES captures(id) ON DELETE CASCADE
);

-- Indexes for common queries
CREATE INDEX IF NOT EXISTS idx_devices_mac ON devices(mac);
CREATE INDEX IF NOT EXISTS idx_devices_capture ON devices(capture_id);
CREATE INDEX IF NOT EXISTS idx_device_ips_device ON device_ips(device_id);
CREATE INDEX IF NOT EXISTS idx_flows_capture ON flows(capture_id);
CREATE INDEX IF NOT EXISTS idx_dns_capture ON dns_domains(capture_id);
