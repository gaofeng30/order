package storefront

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

const settingsQuery = `SELECT
  store_name, store_address, pickup_point, announcement, business_status,
  CONVERT(launch_image_object_key USING utf8mb4), center_x, center_y, width_ratio, aspect_ratio,
  flavor_options_json
FROM storefront_settings
WHERE id = 1
LIMIT 1`

type Repository struct{ db *sql.DB }

func NewRepository(db *sql.DB) *Repository { return &Repository{db: db} }

func (repository *Repository) Get(ctx context.Context) (Settings, error) {
	var settings Settings
	var businessStatus string
	var launchObjectKey sql.NullString
	var centerX, centerY, widthRatio, aspectRatio sql.NullFloat64
	var flavorJSON []byte
	err := repository.db.QueryRowContext(ctx, settingsQuery).Scan(
		&settings.StoreName, &settings.StoreAddress, &settings.PickupPoint, &settings.Announcement, &businessStatus,
		&launchObjectKey, &centerX, &centerY, &widthRatio, &aspectRatio, &flavorJSON,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Settings{}, errors.New("storefront settings missing")
	}
	if err != nil {
		return Settings{}, fmt.Errorf("query storefront settings: %w", err)
	}

	settings.BusinessStatus = BusinessStatus(businessStatus)
	present := 0
	for _, valid := range []bool{launchObjectKey.Valid, centerX.Valid, centerY.Valid, widthRatio.Valid, aspectRatio.Valid} {
		if valid {
			present++
		}
	}
	if present != 0 && present != 5 {
		return Settings{}, errors.New("storefront launch layer invariant failed")
	}
	if present == 5 {
		settings.LaunchLayer = &LaunchLayer{
			ImageObjectKey: launchObjectKey.String, CenterX: centerX.Float64, CenterY: centerY.Float64,
			WidthRatio: widthRatio.Float64, AspectRatio: aspectRatio.Float64,
		}
	}
	if !json.Valid(flavorJSON) || json.Unmarshal(flavorJSON, &settings.Flavors) != nil || !settings.valid() {
		return Settings{}, errors.New("storefront settings invariant failed")
	}
	return settings, nil
}
