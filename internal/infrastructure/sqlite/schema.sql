PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS site_config (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    whatsapp TEXT NOT NULL,
    phone TEXT NOT NULL,
    instagram TEXT NOT NULL,
    hours TEXT NOT NULL,
    address TEXT NOT NULL,
    address_kk TEXT NOT NULL,
    lat REAL NOT NULL,
    lon REAL NOT NULL,
    two_gis_url TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    time_step INTEGER NOT NULL CHECK (time_step IN (15, 30, 60))
);

CREATE TABLE IF NOT EXISTS zones (
    id TEXT PRIMARY KEY,
    display_order INTEGER NOT NULL,
    name TEXT NOT NULL,
    name_kk TEXT NOT NULL,
    size TEXT NOT NULL,
    seats INTEGER,
    description TEXT NOT NULL,
    description_kk TEXT NOT NULL,
    image TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS zone_tariffs (
    zone_id TEXT NOT NULL REFERENCES zones(id) ON DELETE CASCADE,
    day_type TEXT NOT NULL CHECK (day_type IN ('wd', 'we')),
    h1 INTEGER NOT NULL,
    h3 INTEGER NOT NULL,
    h5 INTEGER NOT NULL,
    day INTEGER NOT NULL,
    night INTEGER NOT NULL,
    PRIMARY KEY (zone_id, day_type)
);

CREATE TABLE IF NOT EXISTS zone_specs (
    zone_id TEXT NOT NULL REFERENCES zones(id) ON DELETE CASCADE,
    sort_order INTEGER NOT NULL,
    name TEXT NOT NULL,
    value TEXT NOT NULL,
    PRIMARY KEY (zone_id, sort_order)
);

CREATE TABLE IF NOT EXISTS refresh_sessions (
    token_hash TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    expires_at INTEGER NOT NULL,
    revoked_at INTEGER
);

CREATE INDEX IF NOT EXISTS idx_refresh_sessions_expiry ON refresh_sessions(expires_at);
