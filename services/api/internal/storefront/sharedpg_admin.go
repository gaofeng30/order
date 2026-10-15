package storefront

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strconv"
	"time"
)

type SharedPGAdminApplication struct {
	client     sharedPGExecutor
	confirmed  ObjectConfirmationReader
	receiptNow func() time.Time
}

func NewSharedPGAdminApplication(client sharedPGExecutor, confirmed ObjectConfirmationReader) *SharedPGAdminApplication {
	return &SharedPGAdminApplication{client: client, confirmed: confirmed, receiptNow: time.Now}
}
func (a *SharedPGAdminApplication) Admin(ctx context.Context) (AdminSettings, error) {
	if a == nil || a.client == nil || ctx == nil {
		return AdminSettings{}, ErrAdminUnavailable
	}
	var f struct {
		Outcome  string        `json:"outcome"`
		Settings AdminSettings `json:"settings"`
	}
	if a.client.CallRPC(ctx, "order_storefront_admin_v1", map[string]any{}, &f) != nil || f.Outcome != "SUCCEEDED" || len(f.Settings.MealPeriods) != 2 {
		return AdminSettings{}, ErrAdminUnavailable
	}
	return f.Settings, nil
}
func (a *SharedPGAdminApplication) LaunchLayer(ctx context.Context) (LaunchLayerConfig, error) {
	if a == nil || a.client == nil || ctx == nil {
		return LaunchLayerConfig{}, ErrAdminUnavailable
	}
	var f struct {
		Outcome string  `json:"outcome"`
		Key     *string `json:"launch_key"`
		CenterX *string `json:"center_x"`
		CenterY *string `json:"center_y"`
		Width   *string `json:"width_ratio"`
		Aspect  *string `json:"aspect_ratio"`
	}
	if a.client.CallRPC(ctx, "order_storefront_launch_read_v1", map[string]any{}, &f) != nil || f.Outcome != "SUCCEEDED" {
		return LaunchLayerConfig{}, ErrAdminUnavailable
	}
	present := 0
	for _, valid := range []bool{f.Key != nil, f.CenterX != nil, f.CenterY != nil, f.Width != nil, f.Aspect != nil} {
		if valid {
			present++
		}
	}
	if present == 0 {
		return LaunchLayerConfig{}, nil
	}
	if present != 5 {
		return LaunchLayerConfig{}, ErrAdminUnavailable
	}
	values := make([]float64, 4)
	for i, raw := range []*string{f.CenterX, f.CenterY, f.Width, f.Aspect} {
		v, err := strconv.ParseFloat(*raw, 64)
		if err != nil {
			return LaunchLayerConfig{}, ErrAdminUnavailable
		}
		values[i] = v
	}
	return LaunchLayerConfig{ImageObjectKey: *f.Key, Enabled: true, CenterX: values[0], CenterY: values[1], SizeRatio: values[2], AspectRatio: values[3]}, nil
}

type sharedStorefrontCommandResponse struct {
	Outcome  string  `json:"outcome"`
	Before   *string `json:"before_text"`
	Response *string `json:"response_text"`
}

func sharedStorefrontHash(prefix, value string) string {
	h := sha256.Sum256([]byte(prefix + ":" + value))
	return hex.EncodeToString(h[:])
}
func sharedStorefrontReplay(f sharedStorefrontCommandResponse, digest *[32]byte, out any) error {
	if digest == nil || f.Before == nil || f.Response == nil {
		return ErrAdminUnavailable
	}
	var evidence struct {
		RequestDigest string `json:"request_digest"`
	}
	if !json.Valid([]byte(*f.Before)) || json.Unmarshal([]byte(*f.Before), &evidence) != nil {
		return ErrAdminUnavailable
	}
	stored, err := hex.DecodeString(evidence.RequestDigest)
	if err != nil || len(stored) != len(digest) || subtle.ConstantTimeCompare(stored, digest[:]) != 1 {
		return ErrAdminIdempotencyConflict
	}
	if !json.Valid([]byte(*f.Response)) || json.Unmarshal([]byte(*f.Response), out) != nil {
		return ErrAdminUnavailable
	}
	return nil
}
func sharedStorefrontMeta(meta WriteMeta, request any) (map[string]any, *[32]byte) {
	p := map[string]any{"p_user": strconv.FormatUint(meta.ActorUserID, 10), "p_operation_hash": sharedStorefrontHash("operation", meta.IdempotencyKey), "p_request_id_hash": nil, "p_request_digest": nil, "p_receipt_at": nil}
	if meta.RequestID != "" {
		p["p_request_id_hash"] = sharedStorefrontHash("request", meta.RequestID)
	}
	raw, err := json.Marshal(request)
	if err != nil {
		return p, nil
	}
	digest := sha256.Sum256(raw)
	p["p_request_digest"] = hex.EncodeToString(digest[:])
	return p, &digest
}
func (a *SharedPGAdminApplication) storefrontCommand(ctx context.Context, name string, p map[string]any) (sharedStorefrontCommandResponse, error) {
	var f sharedStorefrontCommandResponse
	if a.client.CallRPC(ctx, name, p, &f) != nil {
		return f, ErrAdminUnavailable
	}
	if f.Outcome == "NEED_MATERIAL" {
		if a.receiptNow == nil {
			return f, ErrAdminUnavailable
		}
		p["p_receipt_at"] = a.receiptNow().UTC().Truncate(time.Microsecond)
		if a.client.CallRPC(ctx, name, p, &f) != nil {
			return f, ErrAdminUnavailable
		}
	}
	switch f.Outcome {
	case "SUCCEEDED", "REPLAY":
		return f, nil
	case "INVALID_INPUT":
		return f, ErrAdminInvalidInput
	case "CONFLICT":
		return f, ErrAdminConflict
	default:
		return f, ErrAdminUnavailable
	}
}
func (a *SharedPGAdminApplication) Configure(ctx context.Context, meta WriteMeta, cmd SettingsCommand) (AdminSettings, error) {
	if len(cmd.MealPeriods) != 2 || cmd.PickupStepMin == 0 {
		return AdminSettings{}, ErrAdminInvalidInput
	}
	if a == nil || a.client == nil || ctx == nil {
		return AdminSettings{}, ErrAdminUnavailable
	}
	p, digest := sharedStorefrontMeta(meta, cmd)
	p["p_store_status_hex"] = hex.EncodeToString([]byte(normalizeStatus(cmd.StoreStatus)))
	p["p_notice_hex"] = hex.EncodeToString([]byte(cmd.Notice))
	p["p_pickup_point_hex"] = hex.EncodeToString([]byte(cmd.PickupPoint))
	p["p_pickup_step_min"] = cmd.PickupStepMin
	meals := make([]map[string]string, 0, len(cmd.MealPeriods))
	for _, m := range cmd.MealPeriods {
		meals = append(meals, map[string]string{"code_hex": hex.EncodeToString([]byte(m.Code)), "cutoff_hex": hex.EncodeToString([]byte(m.CutoffTime)), "pickup_from_hex": hex.EncodeToString([]byte(m.PickupFrom)), "pickup_to_hex": hex.EncodeToString([]byte(m.PickupTo))})
	}
	p["p_meal_periods"] = meals
	dates := append([]ServiceDateConfig(nil), cmd.ServiceDates...)
	sort.Slice(dates, func(i, j int) bool { return dates[i].Date < dates[j].Date })
	dateParams := make([]map[string]string, 0, len(dates))
	for _, d := range dates {
		dateParams = append(dateParams, map[string]string{"date_hex": hex.EncodeToString([]byte(d.Date)), "status_hex": hex.EncodeToString([]byte(d.Status))})
	}
	p["p_service_dates"] = dateParams
	f, err := a.storefrontCommand(ctx, "order_storefront_configure_v1", p)
	if err != nil {
		return AdminSettings{}, err
	}
	var result AdminSettings
	if f.Outcome == "REPLAY" {
		err = sharedStorefrontReplay(f, digest, &result)
	} else if f.Response == nil || json.Unmarshal([]byte(*f.Response), &result) != nil {
		err = ErrAdminUnavailable
	}
	if err != nil {
		return AdminSettings{}, err
	}
	return result, nil
}
func (a *SharedPGAdminApplication) ConfigureLaunchLayer(ctx context.Context, meta WriteMeta, layer *LaunchLayerConfig) (*LaunchLayerConfig, error) {
	if a == nil || a.client == nil || ctx == nil {
		return nil, ErrAdminUnavailable
	}
	p, digest := sharedStorefrontMeta(meta, layer)
	p["p_layer_present"] = layer != nil
	p["p_enabled"] = nil
	p["p_image_key_hex"] = nil
	p["p_image_key_json_hex"] = nil
	p["p_center_x"] = nil
	p["p_center_y"] = nil
	p["p_width_ratio"] = nil
	p["p_aspect_ratio"] = nil
	p["p_confirm_status"] = nil
	if layer != nil {
		p["p_enabled"] = layer.Enabled
	}
	if layer != nil && layer.Enabled {
		p["p_image_key_hex"] = hex.EncodeToString([]byte(layer.ImageObjectKey))
		keyJSON, _ := json.Marshal(layer.ImageObjectKey)
		p["p_image_key_json_hex"] = hex.EncodeToString(keyJSON)
		p["p_center_x"] = strconv.FormatFloat(layer.CenterX, 'g', -1, 64)
		p["p_center_y"] = strconv.FormatFloat(layer.CenterY, 'g', -1, 64)
		p["p_width_ratio"] = strconv.FormatFloat(layer.SizeRatio, 'g', -1, 64)
		p["p_aspect_ratio"] = strconv.FormatFloat(layer.AspectRatio, 'g', -1, 64)
		status := "CONFIRMED"
		if a.confirmed != nil {
			facts, err := a.confirmed.Confirmed(ctx, []string{layer.ImageObjectKey})
			if err != nil {
				status = "UNAVAILABLE"
			} else if !facts[layer.ImageObjectKey] {
				status = "NOT_CONFIRMED"
			}
		}
		p["p_confirm_status"] = status
	}
	f, err := a.storefrontCommand(ctx, "order_storefront_launch_configure_v1", p)
	if err != nil {
		return nil, err
	}
	var result LaunchLayerConfig
	if f.Outcome == "REPLAY" {
		err = sharedStorefrontReplay(f, digest, &result)
	} else if f.Response == nil || json.Unmarshal([]byte(*f.Response), &result) != nil {
		err = ErrAdminUnavailable
	} else if layer != nil && layer.Enabled {
		result = *layer
	}
	if err != nil {
		return nil, err
	}
	return &result, nil
}

var _ AdminApplication = (*SharedPGAdminApplication)(nil)
