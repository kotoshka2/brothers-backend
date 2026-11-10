package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"brothers-backend/internal/application"
	"brothers-backend/internal/domain"
)

func TestRepositorySeedsAndPersistsZoneChanges(t *testing.T) {
	ctx := context.Background()
	repository, err := Open(ctx, filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer repository.Close()

	zones, err := repository.ListZones(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(zones) != 7 || zones[0].ID != "general" {
		t.Fatalf("unexpected seeded zones: count=%d first=%q", len(zones), zones[0].ID)
	}
	config, err := repository.GetConfig(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if config.WhatsApp != "77778300406" || config.Lat != 49.951129 {
		t.Fatalf("unexpected seeded config: %+v", config)
	}
	settings, err := repository.GetSettings(ctx)
	if err != nil || settings.TimeStep != 30 {
		t.Fatalf("unexpected seeded settings: %+v, error: %v", settings, err)
	}

	seats := 2
	zone := domain.Zone{
		ID: "test-room", Name: "Test", NameKK: "Test", Size: "S", Seats: &seats,
		Desc: "Description", DescKK: "Description", Image: "test.jpg",
		Tariff: domain.Tariff{
			Weekdays: domain.Rate{H1: 100, H3: 200, H5: 300, Day: 400, Night: 500},
			Weekends: domain.Rate{H1: 150, H3: 250, H5: 350, Day: 450, Night: 550},
		},
		Specs: []domain.Specification{{Name: "CPU", Value: "Test CPU"}},
	}
	if err := repository.CreateZone(ctx, zone); err != nil {
		t.Fatalf("create zone: %v", err)
	}
	if err := repository.CreateZone(ctx, zone); !errors.Is(err, application.ErrConflict) {
		t.Fatalf("expected duplicate zone conflict, got %v", err)
	}
	created, err := repository.GetZone(ctx, zone.ID)
	if err != nil {
		t.Fatal(err)
	}
	if created.Tariff.Weekends.Night != 550 || len(created.Specs) != 1 || created.Specs[0].Value != "Test CPU" {
		t.Fatalf("zone details were not persisted: %+v", created)
	}

	zone.Name = "Updated"
	zone.Specs = []domain.Specification{{Name: "CPU", Value: "Updated CPU"}}
	if err := repository.UpdateZone(ctx, zone); err != nil {
		t.Fatalf("update zone: %v", err)
	}
	updated, err := repository.GetZone(ctx, zone.ID)
	if err != nil || updated.Name != "Updated" || updated.Specs[0].Value != "Updated CPU" {
		t.Fatalf("zone update was not persisted: %+v, error: %v", updated, err)
	}
	if err := repository.DeleteZone(ctx, zone.ID); err != nil {
		t.Fatalf("delete zone: %v", err)
	}
	if _, err := repository.GetZone(ctx, zone.ID); !errors.Is(err, application.ErrNotFound) {
		t.Fatalf("expected deleted zone to be missing, got %v", err)
	}
}
