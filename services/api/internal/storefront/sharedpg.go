package storefront

import (
	"context"
	"errors"
)

type sharedPGExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}
type SharedPGRepository struct{ client sharedPGExecutor }

func NewSharedPGRepository(client sharedPGExecutor) *SharedPGRepository {
	return &SharedPGRepository{client}
}

var errSharedStorefront = errors.New("shared storefront unavailable")

func (r *SharedPGRepository) Get(ctx context.Context) (Settings, error) {
	if r == nil || r.client == nil || ctx == nil {
		return Settings{}, errSharedStorefront
	}
	var facts struct {
		Outcome     string   `json:"outcome"`
		Settings    Settings `json:"settings"`
		Key         *string  `json:"launch_key"`
		CenterX     *float64 `json:"center_x"`
		CenterY     *float64 `json:"center_y"`
		WidthRatio  *float64 `json:"width_ratio"`
		AspectRatio *float64 `json:"aspect_ratio"`
	}
	if r.client.CallRPC(ctx, "order_storefront_get_v1", map[string]any{}, &facts) != nil || facts.Outcome != "SUCCEEDED" {
		return Settings{}, errSharedStorefront
	}
	present := 0
	for _, valid := range []bool{facts.Key != nil, facts.CenterX != nil, facts.CenterY != nil, facts.WidthRatio != nil, facts.AspectRatio != nil} {
		if valid {
			present++
		}
	}
	if present != 0 && present != 5 {
		return Settings{}, errSharedStorefront
	}
	if present == 5 {
		facts.Settings.LaunchLayer = &LaunchLayer{ImageObjectKey: *facts.Key, CenterX: *facts.CenterX, CenterY: *facts.CenterY, WidthRatio: *facts.WidthRatio, AspectRatio: *facts.AspectRatio}
	}
	if !facts.Settings.valid() {
		return Settings{}, errSharedStorefront
	}
	return facts.Settings, nil
}

var _ Reader = (*SharedPGRepository)(nil)
