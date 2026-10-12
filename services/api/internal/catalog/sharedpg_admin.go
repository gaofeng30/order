package catalog

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
)

type SharedPGAdminApplication struct {
	client     sharedPGExecutor
	images     ImageURLer
	confirmed  ImageConfirmationReader
	receiptNow func() time.Time
}

func NewSharedPGAdminApplication(client sharedPGExecutor, images ImageURLer, confirmed ImageConfirmationReader) *SharedPGAdminApplication {
	return &SharedPGAdminApplication{client: client, images: images, confirmed: confirmed, receiptNow: time.Now}
}
func sharedAdminID(raw string) (uint64, bool) {
	id, err := strconv.ParseUint(raw, 10, 64)
	return id, err == nil && strconv.FormatUint(id, 10) == raw
}
func (a *SharedPGAdminApplication) ListCategories(ctx context.Context) ([]AdminCategory, error) {
	if a == nil || a.client == nil || ctx == nil {
		return nil, ErrAdminUnavailable
	}
	var f struct {
		Outcome    string `json:"outcome"`
		Categories []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			SortOrder uint32 `json:"sort_order"`
			Enabled   bool   `json:"enabled"`
			Count     uint32 `json:"product_count"`
		} `json:"categories"`
	}
	if a.client.CallRPC(ctx, "order_catalog_categories_v1", map[string]any{}, &f) != nil || f.Outcome != "SUCCEEDED" || f.Categories == nil {
		return nil, ErrAdminUnavailable
	}
	result := make([]AdminCategory, 0, len(f.Categories))
	for _, c := range f.Categories {
		id, ok := sharedAdminID(c.ID)
		if !ok {
			return nil, ErrAdminUnavailable
		}
		result = append(result, AdminCategory{ID: id, Name: c.Name, SortOrder: c.SortOrder, Enabled: c.Enabled, ProductCount: c.Count})
	}
	return result, nil
}

type sharedAdminProductFacts struct {
	ID           string          `json:"id"`
	Category     string          `json:"category_id"`
	CategoryName string          `json:"category_name"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	MealPeriod   string          `json:"meal_period"`
	Images       json.RawMessage `json:"images"`
	Price        uint32          `json:"price_cents"`
	Listed       bool            `json:"listed"`
	SoldOut      bool            `json:"sold_out"`
}

func (a *SharedPGAdminApplication) adminProduct(ctx context.Context, f sharedAdminProductFacts) (AdminProduct, error) {
	id, ok := sharedAdminID(f.ID)
	category, cok := sharedAdminID(f.Category)
	if !ok || !cok {
		return AdminProduct{}, ErrAdminUnavailable
	}
	p := AdminProduct{ID: id, CategoryID: category, CategoryName: f.CategoryName, Name: f.Name, Description: f.Description, MealPeriod: f.MealPeriod, PriceCents: f.Price, Listed: f.Listed, SoldOut: f.SoldOut}
	var stored []struct {
		ObjectKey string `json:"object_key"`
	}
	if !json.Valid(f.Images) || json.Unmarshal(f.Images, &stored) != nil || len(stored) > 3 {
		return AdminProduct{}, ErrAdminUnavailable
	}
	seen := map[string]bool{}
	for i, item := range stored {
		if item.ObjectKey == "" || seen[item.ObjectKey] || a.images == nil {
			return AdminProduct{}, ErrAdminUnavailable
		}
		seen[item.ObjectKey] = true
		url, err := a.images.PublicURL(ctx, item.ObjectKey)
		if err != nil {
			return AdminProduct{}, ErrAdminUnavailable
		}
		p.Images = append(p.Images, AdminImage{ObjectKey: item.ObjectKey, URL: url, SortOrder: uint8(i)})
	}
	return p, nil
}
func (a *SharedPGAdminApplication) ListProducts(ctx context.Context, q AdminQuery) ([]AdminProduct, error) {
	if q.ServiceDate == "" {
		return nil, ErrAdminInvalidInput
	}
	if a == nil || a.client == nil || ctx == nil {
		return nil, ErrAdminUnavailable
	}
	var f struct {
		Outcome  string                    `json:"outcome"`
		Products []sharedAdminProductFacts `json:"products"`
	}
	if a.client.CallRPC(ctx, "order_catalog_products_v1", map[string]any{"p_service_date": q.ServiceDate}, &f) != nil || f.Outcome != "SUCCEEDED" || f.Products == nil {
		return nil, ErrAdminUnavailable
	}
	result := make([]AdminProduct, 0, len(f.Products))
	for _, pf := range f.Products {
		p, err := a.adminProduct(ctx, pf)
		if err != nil {
			return nil, err
		}
		result = append(result, p)
	}
	return result, nil
}
func (a *SharedPGAdminApplication) GetProduct(ctx context.Context, id uint64, q AdminQuery) (AdminProduct, error) {
	if id == 0 || q.ServiceDate == "" {
		return AdminProduct{}, ErrAdminInvalidInput
	}
	if a == nil || a.client == nil || ctx == nil {
		return AdminProduct{}, ErrAdminUnavailable
	}
	var f struct {
		Outcome string                   `json:"outcome"`
		Product *sharedAdminProductFacts `json:"product"`
	}
	if a.client.CallRPC(ctx, "order_catalog_product_v1", map[string]any{"p_product": strconv.FormatUint(id, 10), "p_service_date": q.ServiceDate}, &f) != nil || f.Outcome != "SUCCEEDED" {
		return AdminProduct{}, ErrAdminUnavailable
	}
	if f.Product == nil {
		return AdminProduct{}, ErrAdminNotFound
	}
	return a.adminProduct(ctx, *f.Product)
}

type sharedCatalogCommandResponse struct {
	Outcome  string  `json:"outcome"`
	Before   *string `json:"before_text"`
	Response *string `json:"response_text"`
}

func sharedCatalogHash(prefix, value string) string {
	h := sha256.Sum256([]byte(prefix + ":" + value))
	return hex.EncodeToString(h[:])
}
func sharedCatalogReplay(f sharedCatalogCommandResponse, digest [32]byte) (CatalogResult, error) {
	if f.Before == nil || f.Response == nil {
		return CatalogResult{}, ErrAdminUnavailable
	}
	var evidence struct {
		RequestDigest string `json:"request_digest"`
	}
	if !json.Valid([]byte(*f.Before)) || json.Unmarshal([]byte(*f.Before), &evidence) != nil {
		return CatalogResult{}, ErrAdminUnavailable
	}
	stored, err := hex.DecodeString(evidence.RequestDigest)
	if err != nil || len(stored) != len(digest) || subtle.ConstantTimeCompare(stored, digest[:]) != 1 {
		return CatalogResult{}, ErrAdminIdempotencyConflict
	}
	var result CatalogResult
	if !json.Valid([]byte(*f.Response)) || json.Unmarshal([]byte(*f.Response), &result) != nil {
		return CatalogResult{}, ErrAdminUnavailable
	}
	return result, nil
}
func (a *SharedPGAdminApplication) Execute(ctx context.Context, meta WriteMeta, cmd CatalogCommand) (CatalogResult, error) {
	if a == nil || a.client == nil || ctx == nil {
		return CatalogResult{}, ErrAdminUnavailable
	}
	raw, err := json.Marshal(cmd)
	if err != nil {
		return CatalogResult{}, ErrAdminUnavailable
	}
	digest := sha256.Sum256(raw)
	parameters := map[string]any{"p_user": strconv.FormatUint(meta.ActorUserID, 10), "p_kind": string(cmd.Kind), "p_category": strconv.FormatUint(cmd.CategoryID, 10), "p_product": strconv.FormatUint(cmd.ProductID, 10), "p_name_present": nil, "p_name_hex": nil, "p_description_hex": nil, "p_price_cents": nil, "p_meal_period_hex": nil, "p_images": nil, "p_enabled": nil, "p_listed": nil, "p_service_date_hex": nil, "p_sold_out": nil, "p_ordered_ids": nil, "p_confirm_status": nil, "p_operation_hash": sharedCatalogHash("operation", meta.IdempotencyKey), "p_request_id_hash": nil, "p_request_digest": hex.EncodeToString(digest[:]), "p_receipt_at": nil}
	if meta.RequestID != "" {
		parameters["p_request_id_hash"] = sharedCatalogHash("request", meta.RequestID)
	}
	if cmd.Kind == CommandCreateCategory || cmd.Kind == CommandUpdateCategory || cmd.Kind == CommandCreateProduct || cmd.Kind == CommandUpdateProduct {
		name, _, _ := catalogName(cmd.Name)
		parameters["p_name_present"] = cmd.Name != ""
		parameters["p_name_hex"] = hex.EncodeToString([]byte(name))
	}
	if cmd.Kind == CommandUpdateCategory {
		parameters["p_enabled"] = cmd.Enabled
	}
	if cmd.Kind == CommandCreateProduct || cmd.Kind == CommandUpdateProduct {
		parameters["p_description_hex"] = hex.EncodeToString([]byte(cmd.Description))
		parameters["p_price_cents"] = int64(cmd.PriceCents)
		parameters["p_meal_period_hex"] = hex.EncodeToString([]byte(cmd.MealPeriod))
		if cmd.Images != nil {
			images := make([]map[string]any, 0, len(cmd.Images))
			for _, image := range cmd.Images {
				keyJSON, _ := json.Marshal(image.ObjectKey)
				urlJSON, _ := json.Marshal(image.URL)
				images = append(images, map[string]any{"object_key_hex": hex.EncodeToString([]byte(image.ObjectKey)), "object_key_json_hex": hex.EncodeToString(keyJSON), "url_json_hex": hex.EncodeToString(urlJSON), "sort_order": image.SortOrder})
			}
			parameters["p_images"] = images
		}
		status := "CONFIRMED"
		if len(cmd.Images) > 0 && a.confirmed != nil {
			keys := make([]string, 0, len(cmd.Images))
			for _, image := range cmd.Images {
				keys = append(keys, image.ObjectKey)
			}
			confirmed, err := a.confirmed.Confirmed(ctx, keys)
			if err != nil {
				status = "UNAVAILABLE"
			} else {
				for _, key := range keys {
					if !confirmed[key] {
						status = "NOT_CONFIRMED"
						break
					}
				}
			}
		}
		parameters["p_confirm_status"] = status
	}
	if cmd.Kind == CommandProductStatus {
		parameters["p_listed"] = cmd.Listed
	}
	if cmd.Kind == CommandProductSoldOut {
		parameters["p_service_date_hex"] = hex.EncodeToString([]byte(cmd.ServiceDate))
		parameters["p_sold_out"] = cmd.SoldOut
	}
	if cmd.Kind == CommandReorderCategory || cmd.Kind == CommandReorderProducts {
		ids := make([]string, 0, len(cmd.OrderedIDs))
		for _, id := range cmd.OrderedIDs {
			ids = append(ids, strconv.FormatUint(id, 10))
		}
		parameters["p_ordered_ids"] = ids
	}
	var facts sharedCatalogCommandResponse
	if a.client.CallRPC(ctx, "order_catalog_command_v1", parameters, &facts) != nil {
		return CatalogResult{}, ErrAdminUnavailable
	}
	if facts.Outcome == "NEED_MATERIAL" {
		if a.receiptNow == nil {
			return CatalogResult{}, ErrAdminUnavailable
		}
		parameters["p_receipt_at"] = a.receiptNow().UTC().Truncate(time.Microsecond)
		if a.client.CallRPC(ctx, "order_catalog_command_v1", parameters, &facts) != nil {
			return CatalogResult{}, ErrAdminUnavailable
		}
	}
	switch facts.Outcome {
	case "REPLAY":
		return sharedCatalogReplay(facts, digest)
	case "SUCCEEDED":
		if facts.Response == nil {
			return CatalogResult{}, ErrAdminUnavailable
		}
		var result CatalogResult
		if json.Unmarshal([]byte(*facts.Response), &result) != nil {
			return CatalogResult{}, ErrAdminUnavailable
		}
		if result.Product != nil && (cmd.Kind == CommandCreateProduct || cmd.Kind == CommandUpdateProduct) {
			result.Product.Images = cmd.Images
		}
		return result, nil
	case "INVALID_INPUT":
		return CatalogResult{}, ErrAdminInvalidInput
	case "NOT_FOUND":
		return CatalogResult{}, ErrAdminNotFound
	case "CONFLICT":
		return CatalogResult{}, ErrAdminConflict
	default:
		return CatalogResult{}, ErrAdminUnavailable
	}
}

var _ AdminApplication = (*SharedPGAdminApplication)(nil)
