package storefront

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

const settingsQuery = `SELECT
  store_name, store_address, pickup_point, announcement, business_status,
  launch_png_url, center_x, center_y, width_ratio, aspect_ratio
FROM storefront_settings
WHERE id = 1
LIMIT 1`

// Repository reads the singleton public storefront settings from MySQL.
type Repository struct {
	db *sql.DB
}

// NewRepository constructs a storefront settings repository over the shared pool.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

// Get returns the configured singleton or fails closed for every unavailable or invalid state.
func (repository *Repository) Get(ctx context.Context) (Settings, error) {
	var settings Settings
	var businessStatus string
	var launchPNGURL sql.NullString
	var centerX, centerY, widthRatio, aspectRatio sql.NullFloat64
	err := repository.db.QueryRowContext(ctx, settingsQuery).Scan(
		&settings.StoreName, &settings.StoreAddress, &settings.PickupPoint, &settings.Announcement, &businessStatus,
		&launchPNGURL, &centerX, &centerY, &widthRatio, &aspectRatio,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Settings{}, errors.New("storefront settings missing")
	}
	if err != nil {
		return Settings{}, fmt.Errorf("query storefront settings: %w", err)
	}

	settings.BusinessStatus = BusinessStatus(businessStatus)
	launchFields := []bool{launchPNGURL.Valid, centerX.Valid, centerY.Valid, widthRatio.Valid, aspectRatio.Valid}
	present := 0
	for _, valid := range launchFields {
		if valid {
			present++
		}
	}
	if present != 0 && present != len(launchFields) {
		return Settings{}, errors.New("storefront launch layer invariant failed")
	}
	if present == len(launchFields) {
		settings.LaunchLayer = &LaunchLayer{
			PNGURL: launchPNGURL.String, CenterX: centerX.Float64, CenterY: centerY.Float64,
			WidthRatio: widthRatio.Float64, AspectRatio: aspectRatio.Float64,
		}
	}
	if !settings.valid() {
		return Settings{}, errors.New("storefront settings invariant failed")
	}
	return settings, nil
}
