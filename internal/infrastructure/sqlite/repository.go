package sqlite

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"brothers-backend/internal/domain"

	_ "modernc.org/sqlite"
)

//go:embed schema.sql
var schema string

type Repository struct {
	db *sql.DB
}

func Open(ctx context.Context, path string) (*Repository, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect to sqlite database: %w", err)
	}
	if _, err := db.ExecContext(ctx, schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize sqlite schema: %w", err)
	}
	repository := &Repository{db: db}
	if err := repository.seedDefaults(ctx); err != nil {
		db.Close()
		return nil, err
	}
	return repository, nil
}

func (r *Repository) Close() error {
	return r.db.Close()
}

func (r *Repository) seedDefaults(ctx context.Context) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin default data transaction: %w", err)
	}
	defer tx.Rollback()

	config := domain.DefaultConfig()
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO site_config (id, whatsapp, phone, instagram, hours, address, address_kk, lat, lon, two_gis_url)
		VALUES (1, ?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
		config.WhatsApp, config.Phone, config.Instagram, config.Hours, config.Address,
		config.AddressKK, config.Lat, config.Lon, config.TwoGISURL); err != nil {
		return fmt.Errorf("seed site config: %w", err)
	}
	settings := domain.DefaultSettings()
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO settings (id, time_step) VALUES (1, ?) ON CONFLICT(id) DO NOTHING`,
		settings.TimeStep); err != nil {
		return fmt.Errorf("seed settings: %w", err)
	}
	for order, zone := range domain.DefaultZones() {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO zones (id, display_order, name, name_kk, size, seats, description, description_kk, image)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(id) DO NOTHING`,
			zone.ID, order, zone.Name, zone.NameKK, zone.Size, zone.Seats, zone.Desc,
			zone.DescKK, zone.Image); err != nil {
			return fmt.Errorf("seed zone %q: %w", zone.ID, err)
		}
		var count int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM zone_tariffs WHERE zone_id = ?`, zone.ID).Scan(&count); err != nil {
			return fmt.Errorf("check seeded tariffs for zone %q: %w", zone.ID, err)
		}
		if count == 0 {
			if err := insertTariffs(ctx, tx, zone.ID, zone.Tariff); err != nil {
				return fmt.Errorf("seed tariffs for zone %q: %w", zone.ID, err)
			}
		}
		if err := insertSpecsIfEmpty(ctx, tx, zone.ID, zone.Specs); err != nil {
			return fmt.Errorf("seed specifications for zone %q: %w", zone.ID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit default data: %w", err)
	}
	return nil
}

func (r *Repository) GetConfig(ctx context.Context) (domain.Config, error) {
	var config domain.Config
	err := r.db.QueryRowContext(ctx, `
		SELECT whatsapp, phone, instagram, hours, address, address_kk, lat, lon, two_gis_url
		FROM site_config WHERE id = 1`).Scan(
		&config.WhatsApp, &config.Phone, &config.Instagram, &config.Hours,
		&config.Address, &config.AddressKK, &config.Lat, &config.Lon, &config.TwoGISURL)
	if err != nil {
		return domain.Config{}, fmt.Errorf("get config: %w", err)
	}
	return config, nil
}

func (r *Repository) UpdateConfig(ctx context.Context, config domain.Config) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE site_config SET whatsapp = ?, phone = ?, instagram = ?, hours = ?,
		address = ?, address_kk = ?, lat = ?, lon = ?, two_gis_url = ? WHERE id = 1`,
		config.WhatsApp, config.Phone, config.Instagram, config.Hours, config.Address,
		config.AddressKK, config.Lat, config.Lon, config.TwoGISURL)
	if err != nil {
		return fmt.Errorf("update config: %w", err)
	}
	if err := requireAffected(result); err != nil {
		return fmt.Errorf("update config: %w", err)
	}
	return nil
}

func (r *Repository) GetSettings(ctx context.Context) (domain.Settings, error) {
	var settings domain.Settings
	if err := r.db.QueryRowContext(ctx, `SELECT time_step FROM settings WHERE id = 1`).
		Scan(&settings.TimeStep); err != nil {
		return domain.Settings{}, fmt.Errorf("get settings: %w", err)
	}
	return settings, nil
}

func (r *Repository) UpdateSettings(ctx context.Context, settings domain.Settings) error {
	result, err := r.db.ExecContext(ctx, `UPDATE settings SET time_step = ? WHERE id = 1`, settings.TimeStep)
	if err != nil {
		return fmt.Errorf("update settings: %w", err)
	}
	if err := requireAffected(result); err != nil {
		return fmt.Errorf("update settings: %w", err)
	}
	return nil
}

func (r *Repository) ListZones(ctx context.Context) ([]domain.Zone, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM zones ORDER BY display_order, id`)
	if err != nil {
		return nil, fmt.Errorf("list zones: %w", err)
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("read zone id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate zone ids: %w", err)
	}
	zones := make([]domain.Zone, 0, len(ids))
	for _, id := range ids {
		zone, err := r.GetZone(ctx, id)
		if err != nil {
			return nil, err
		}
		zones = append(zones, zone)
	}
	return zones, nil
}

func (r *Repository) GetZone(ctx context.Context, id string) (domain.Zone, error) {
	var zone domain.Zone
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, name_kk, size, seats, description, description_kk, image
		FROM zones WHERE id = ?`, id).Scan(
		&zone.ID, &zone.Name, &zone.NameKK, &zone.Size, &zone.Seats,
		&zone.Desc, &zone.DescKK, &zone.Image)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Zone{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Zone{}, fmt.Errorf("get zone %q: %w", id, err)
	}
	if err := readTariffs(ctx, r.db, id, &zone.Tariff); err != nil {
		return domain.Zone{}, err
	}
	specs, err := readSpecs(ctx, r.db, id)
	if err != nil {
		return domain.Zone{}, err
	}
	zone.Specs = specs
	return zone, nil
}

func (r *Repository) CreateZone(ctx context.Context, zone domain.Zone) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create zone transaction: %w", err)
	}
	defer tx.Rollback()
	var exists int
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM zones WHERE id = ?)`, zone.ID).Scan(&exists); err != nil {
		return fmt.Errorf("check existing zone %q: %w", zone.ID, err)
	}
	if exists != 0 {
		return domain.ErrConflict
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO zones (id, display_order, name, name_kk, size, seats, description, description_kk, image)
		VALUES (?, COALESCE((SELECT MAX(display_order) + 1 FROM zones), 0), ?, ?, ?, ?, ?, ?, ?)`,
		zone.ID, zone.Name, zone.NameKK, zone.Size, zone.Seats, zone.Desc, zone.DescKK, zone.Image); err != nil {
		return fmt.Errorf("insert zone %q: %w", zone.ID, err)
	}
	if err := replaceZoneDetails(ctx, tx, zone); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create zone: %w", err)
	}
	return nil
}

func (r *Repository) UpdateZone(ctx context.Context, zone domain.Zone) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update zone transaction: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
		UPDATE zones SET name = ?, name_kk = ?, size = ?, seats = ?, description = ?,
		description_kk = ?, image = ? WHERE id = ?`,
		zone.Name, zone.NameKK, zone.Size, zone.Seats, zone.Desc, zone.DescKK, zone.Image, zone.ID)
	if err != nil {
		return fmt.Errorf("update zone %q: %w", zone.ID, err)
	}
	if err := requireAffected(result); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM zone_tariffs WHERE zone_id = ?`, zone.ID); err != nil {
		return fmt.Errorf("replace tariffs for zone %q: %w", zone.ID, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM zone_specs WHERE zone_id = ?`, zone.ID); err != nil {
		return fmt.Errorf("replace specifications for zone %q: %w", zone.ID, err)
	}
	if err := replaceZoneDetails(ctx, tx, zone); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update zone: %w", err)
	}
	return nil
}

func (r *Repository) DeleteZone(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM zones WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete zone %q: %w", id, err)
	}
	if err := requireAffected(result); err != nil {
		return err
	}
	return nil
}

func (r *Repository) CreateRefreshSession(ctx context.Context, tokenHash, username string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO refresh_sessions (token_hash, username, expires_at) VALUES (?, ?, ?)`,
		tokenHash, username, expiresAt.Unix())
	if err != nil {
		return fmt.Errorf("insert refresh session: %w", err)
	}
	return nil
}

func (r *Repository) RotateRefreshSession(ctx context.Context, oldHash, newHash, username string, expiresAt time.Time) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin refresh rotation: %w", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `
		UPDATE refresh_sessions SET revoked_at = ?
		WHERE token_hash = ? AND username = ? AND revoked_at IS NULL AND expires_at > ?`,
		time.Now().Unix(), oldHash, username, time.Now().Unix())
	if err != nil {
		return fmt.Errorf("consume refresh session: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check refresh session: %w", err)
	}
	if rows != 1 {
		return domain.ErrUnauthorized
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO refresh_sessions (token_hash, username, expires_at) VALUES (?, ?, ?)`,
		newHash, username, expiresAt.Unix()); err != nil {
		return fmt.Errorf("insert rotated refresh session: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit refresh rotation: %w", err)
	}
	return nil
}

func (r *Repository) RevokeRefreshSession(ctx context.Context, tokenHash, username string) error {
	result, err := r.db.ExecContext(ctx, `
		UPDATE refresh_sessions SET revoked_at = ?
		WHERE token_hash = ? AND username = ? AND revoked_at IS NULL`,
		time.Now().Unix(), tokenHash, username)
	if err != nil {
		return fmt.Errorf("revoke refresh session: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("check refresh session: %w", err)
	}
	if rows != 1 {
		return domain.ErrUnauthorized
	}
	return nil
}

func replaceZoneDetails(ctx context.Context, tx *sql.Tx, zone domain.Zone) error {
	if err := insertTariffs(ctx, tx, zone.ID, zone.Tariff); err != nil {
		return fmt.Errorf("insert tariffs for zone %q: %w", zone.ID, err)
	}
	for order, spec := range zone.Specs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO zone_specs (zone_id, sort_order, name, value) VALUES (?, ?, ?, ?)`,
			zone.ID, order, spec.Name, spec.Value); err != nil {
			return fmt.Errorf("insert specification for zone %q: %w", zone.ID, err)
		}
	}
	return nil
}

func insertTariffs(ctx context.Context, tx *sql.Tx, zoneID string, tariff domain.Tariff) error {
	for _, entry := range []struct {
		dayType string
		rate    domain.Rate
	}{{"wd", tariff.Weekdays}, {"we", tariff.Weekends}} {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO zone_tariffs (zone_id, day_type, h1, h3, h5, day, night)
			VALUES (?, ?, ?, ?, ?, ?, ?)`,
			zoneID, entry.dayType, entry.rate.H1, entry.rate.H3, entry.rate.H5,
			entry.rate.Day, entry.rate.Night); err != nil {
			return err
		}
	}
	return nil
}

func insertSpecsIfEmpty(ctx context.Context, tx *sql.Tx, zoneID string, specs []domain.Specification) error {
	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM zone_specs WHERE zone_id = ?`, zoneID).Scan(&count); err != nil {
		return err
	}
	if count != 0 {
		return nil
	}
	for order, spec := range specs {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO zone_specs (zone_id, sort_order, name, value) VALUES (?, ?, ?, ?)`,
			zoneID, order, spec.Name, spec.Value); err != nil {
			return err
		}
	}
	return nil
}

func readTariffs(ctx context.Context, db *sql.DB, zoneID string, tariff *domain.Tariff) error {
	rows, err := db.QueryContext(ctx, `SELECT day_type, h1, h3, h5, day, night FROM zone_tariffs WHERE zone_id = ?`, zoneID)
	if err != nil {
		return fmt.Errorf("read tariffs for zone %q: %w", zoneID, err)
	}
	defer rows.Close()
	found := 0
	for rows.Next() {
		var dayType string
		var rate domain.Rate
		if err := rows.Scan(&dayType, &rate.H1, &rate.H3, &rate.H5, &rate.Day, &rate.Night); err != nil {
			return fmt.Errorf("scan tariffs for zone %q: %w", zoneID, err)
		}
		switch dayType {
		case "wd":
			tariff.Weekdays = rate
		case "we":
			tariff.Weekends = rate
		}
		found++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate tariffs for zone %q: %w", zoneID, err)
	}
	if found != 2 {
		return fmt.Errorf("zone %q has incomplete tariff data", zoneID)
	}
	return nil
}

func readSpecs(ctx context.Context, db *sql.DB, zoneID string) ([]domain.Specification, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT name, value FROM zone_specs WHERE zone_id = ? ORDER BY sort_order`, zoneID)
	if err != nil {
		return nil, fmt.Errorf("read specifications for zone %q: %w", zoneID, err)
	}
	defer rows.Close()
	specs := make([]domain.Specification, 0)
	for rows.Next() {
		var spec domain.Specification
		if err := rows.Scan(&spec.Name, &spec.Value); err != nil {
			return nil, fmt.Errorf("scan specifications for zone %q: %w", zoneID, err)
		}
		specs = append(specs, spec)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate specifications for zone %q: %w", zoneID, err)
	}
	return specs, nil
}

func requireAffected(result sql.Result) error {
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return domain.ErrNotFound
	}
	return nil
}
