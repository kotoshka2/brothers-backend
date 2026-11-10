package application

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"brothers-backend/internal/domain"
)

var (
	ErrNotFound = domain.ErrNotFound
	ErrConflict = domain.ErrConflict
	ErrInvalid  = domain.ErrInvalid
)

type ContentRepository interface {
	GetConfig(context.Context) (domain.Config, error)
	UpdateConfig(context.Context, domain.Config) error
	GetSettings(context.Context) (domain.Settings, error)
	UpdateSettings(context.Context, domain.Settings) error
	ListZones(context.Context) ([]domain.Zone, error)
	GetZone(context.Context, string) (domain.Zone, error)
	CreateZone(context.Context, domain.Zone) error
	UpdateZone(context.Context, domain.Zone) error
	DeleteZone(context.Context, string) error
}

type ContentService struct {
	repository ContentRepository
}

func NewContentService(repository ContentRepository) *ContentService {
	return &ContentService{repository: repository}
}

func (s *ContentService) GetConfig(ctx context.Context) (domain.Config, error) {
	return s.repository.GetConfig(ctx)
}

func (s *ContentService) UpdateConfig(ctx context.Context, config domain.Config) error {
	if strings.TrimSpace(config.WhatsApp) == "" || strings.TrimSpace(config.Phone) == "" ||
		strings.TrimSpace(config.Address) == "" || strings.TrimSpace(config.AddressKK) == "" ||
		strings.TrimSpace(config.Hours) == "" || config.Lat < -90 || config.Lat > 90 ||
		config.Lon < -180 || config.Lon > 180 {
		return fmt.Errorf("%w: invalid config values", ErrInvalid)
	}
	return s.repository.UpdateConfig(ctx, config)
}

func (s *ContentService) GetSettings(ctx context.Context) (domain.Settings, error) {
	return s.repository.GetSettings(ctx)
}

func (s *ContentService) UpdateSettings(ctx context.Context, settings domain.Settings) error {
	if settings.TimeStep != 15 && settings.TimeStep != 30 && settings.TimeStep != 60 {
		return fmt.Errorf("%w: timeStep must be 15, 30, or 60", ErrInvalid)
	}
	return s.repository.UpdateSettings(ctx, settings)
}

func (s *ContentService) ListZones(ctx context.Context) ([]domain.Zone, error) {
	return s.repository.ListZones(ctx)
}

func (s *ContentService) GetZone(ctx context.Context, id string) (domain.Zone, error) {
	return s.repository.GetZone(ctx, id)
}

func (s *ContentService) CreateZone(ctx context.Context, zone domain.Zone) error {
	if err := validateZone(zone); err != nil {
		return err
	}
	return s.repository.CreateZone(ctx, zone)
}

func (s *ContentService) UpdateZone(ctx context.Context, zone domain.Zone) error {
	if err := validateZone(zone); err != nil {
		return err
	}
	return s.repository.UpdateZone(ctx, zone)
}

func (s *ContentService) DeleteZone(ctx context.Context, id string) error {
	return s.repository.DeleteZone(ctx, id)
}

var zoneIDPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func validateZone(zone domain.Zone) error {
	if !zoneIDPattern.MatchString(zone.ID) || strings.TrimSpace(zone.Name) == "" ||
		strings.TrimSpace(zone.NameKK) == "" || strings.TrimSpace(zone.Size) == "" ||
		strings.TrimSpace(zone.Desc) == "" || strings.TrimSpace(zone.DescKK) == "" {
		return fmt.Errorf("%w: required zone fields are missing or invalid", ErrInvalid)
	}
	if zone.Seats != nil && *zone.Seats <= 0 {
		return fmt.Errorf("%w: seats must be positive or null", ErrInvalid)
	}
	rates := []int{
		zone.Tariff.Weekdays.H1, zone.Tariff.Weekdays.H3, zone.Tariff.Weekdays.H5,
		zone.Tariff.Weekdays.Day, zone.Tariff.Weekdays.Night, zone.Tariff.Weekends.H1,
		zone.Tariff.Weekends.H3, zone.Tariff.Weekends.H5, zone.Tariff.Weekends.Day,
		zone.Tariff.Weekends.Night,
	}
	for _, rate := range rates {
		if rate < 0 {
			return fmt.Errorf("%w: tariff values cannot be negative", ErrInvalid)
		}
	}
	for _, spec := range zone.Specs {
		if strings.TrimSpace(spec.Name) == "" {
			return fmt.Errorf("%w: specification name cannot be empty", ErrInvalid)
		}
	}
	return nil
}
