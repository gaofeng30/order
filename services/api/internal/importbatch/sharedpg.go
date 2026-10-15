package importbatch

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"time"
)

type sharedPGExecutor interface {
	CallRPC(context.Context, string, map[string]any, any) error
}

type SharedPGApplication struct {
	confirmed       ImageConfirmationReader
	client          sharedPGExecutor
	now, receiptNow func() time.Time
	token           func() (string, error)
}

func NewSharedPGApplication(client sharedPGExecutor, confirmed ImageConfirmationReader) *SharedPGApplication {
	return &SharedPGApplication{client: client, confirmed: confirmed, now: time.Now, receiptNow: time.Now, token: randomToken}
}

type importRPCResult struct {
	Outcome    string     `json:"outcome"`
	Before     *string    `json:"before_text"`
	Response   *string    `json:"response_text"`
	Plan       *string    `json:"plan_text"`
	PlanDigest string     `json:"plan_digest"`
	Kind       Kind       `json:"kind"`
	ObservedAt *time.Time `json:"observed_at"`
}

func importHash(prefix, value string) string {
	h := sha256.Sum256([]byte(prefix + ":" + value))
	return hex.EncodeToString(h[:])
}
func importDigest(value any) string {
	raw, _ := json.Marshal(value)
	h := sha256.Sum256(raw)
	return hex.EncodeToString(h[:])
}
func importHex(value string) string { return hex.EncodeToString([]byte(value)) }
func importReceiptParameters(meta WriteMeta, request any) map[string]any {
	p := map[string]any{"p_user": strconv.FormatUint(meta.ActorUserID, 10), "p_operation_hash": importHash("operation", meta.IdempotencyKey), "p_request_id_hash": nil, "p_request_digest": importDigest(request), "p_receipt_at": nil}
	if meta.RequestID != "" {
		p["p_request_id_hash"] = importHash("request", meta.RequestID)
	}
	return p
}
func importReplay(f importRPCResult, digest string, out any) error {
	if f.Before == nil || f.Response == nil {
		return ErrUnavailable
	}
	var evidence struct {
		Digest string `json:"request_digest"`
	}
	if json.Unmarshal([]byte(*f.Before), &evidence) != nil {
		return ErrUnavailable
	}
	stored, e := hex.DecodeString(evidence.Digest)
	expected, _ := hex.DecodeString(digest)
	if e != nil || len(stored) != 32 || subtle.ConstantTimeCompare(stored, expected) != 1 {
		return ErrIdempotencyConflict
	}
	if json.Unmarshal([]byte(*f.Response), out) != nil {
		return ErrUnavailable
	}
	return nil
}
func importOutcome(outcome string) error {
	switch outcome {
	case "INVALID_INPUT":
		return ErrInvalidInput
	case "PREVIEW_EXPIRED":
		return ErrPreviewExpired
	case "IDEMPOTENCY_CONFLICT":
		return ErrIdempotencyConflict
	default:
		return ErrUnavailable
	}
}
func (a *SharedPGApplication) Preview(ctx context.Context, meta WriteMeta, kind Kind, file XLSX) (Preview, error) {
	rows, err := ParseRows(file.Bytes, MaxRows(kind))
	if err != nil {
		return Preview{}, err
	}
	if len(rows) < 2 {
		return Preview{}, ErrInvalidTemplate
	}
	header, plan, err := planHeader(kind, rows)
	if err != nil {
		return Preview{}, err
	}
	if a == nil || a.client == nil || ctx == nil {
		return Preview{}, ErrUnavailable
	}
	var facts struct {
		Outcome    string    `json:"outcome"`
		Products   []*string `json:"products"`
		Categories []*string `json:"categories"`
		Staff      []*string `json:"staff"`
	}
	if a.client.CallRPC(ctx, "order_import_preview_facts_v1", map[string]any{"p_kind": kind}, &facts) != nil || facts.Outcome != "SUCCEEDED" {
		return Preview{}, ErrUnavailable
	}
	decode := func(values []*string) (map[string]bool, error) {
		if values == nil {
			return nil, ErrUnavailable
		}
		m := map[string]bool{}
		for _, v := range values {
			if v == nil {
				return nil, ErrUnavailable
			}
			raw, e := hex.DecodeString(*v)
			if e != nil {
				return nil, ErrUnavailable
			}
			m[string(raw)] = true
		}
		return m, nil
	}
	products, categories, staff := map[string]bool{}, map[string]bool{}, map[string]bool{}
	if kind == Product {
		products, err = decode(facts.Products)
		if err == nil {
			categories, err = decode(facts.Categories)
		}
	} else {
		staff, err = decode(facts.Staff)
	}
	if err != nil {
		return Preview{}, err
	}
	plan, err = planRows(kind, rows, header, plan, products, categories, staff)
	if err != nil {
		return Preview{}, err
	}
	if err := attachImages(ctx, &plan, file.Images, a.confirmed); err != nil {
		return Preview{}, err
	}
	if a.token == nil {
		return Preview{}, ErrUnavailable
	}
	token, err := a.token()
	if err != nil {
		return Preview{}, ErrUnavailable
	}
	if a.now == nil {
		return Preview{}, ErrUnavailable
	}
	captured := a.now().UTC().Truncate(time.Microsecond)
	out := planPreview(token, plan, captured.Add(15*time.Minute))
	tokenHash := sha256.Sum256([]byte(token))
	fileDigest := sha256.Sum256(file.Bytes)
	request := struct {
		Kind   Kind
		Digest [32]byte
		Images map[uint32][]ProductImage `json:",omitempty"`
	}{kind, fileDigest, file.Images}
	p := importReceiptParameters(meta, request)
	planRaw, _ := json.Marshal(plan)
	responseRaw, _ := json.Marshal(out)
	p["p_kind"] = kind
	p["p_file_digest"] = hex.EncodeToString(fileDigest[:])
	p["p_file_size"] = len(file.Bytes)
	p["p_row_count"] = len(rows) - 1
	p["p_token_hash"] = hex.EncodeToString(tokenHash[:])
	p["p_plan_json"] = string(planRaw)
	p["p_response_json"] = string(responseRaw)
	p["p_captured_at"] = captured
	var f importRPCResult
	if a.client.CallRPC(ctx, "order_import_preview_v1", p, &f) != nil {
		return Preview{}, ErrUnavailable
	}
	if f.Outcome == "NEED_RECEIPT" {
		if a.receiptNow == nil {
			return Preview{}, ErrUnavailable
		}
		p["p_receipt_at"] = a.receiptNow().UTC().Truncate(time.Microsecond)
		if a.client.CallRPC(ctx, "order_import_preview_v1", p, &f) != nil {
			return Preview{}, ErrUnavailable
		}
	}
	switch f.Outcome {
	case "SUCCEEDED":
		return out, nil
	case "REPLAY":
		var replay Preview
		err = importReplay(f, p["p_request_digest"].(string), &replay)
		return replay, err
	default:
		return Preview{}, importOutcome(f.Outcome)
	}
}

type importMaterial struct {
	Status     string                   `json:"status"`
	Rows       []importMaterialRow      `json:"rows"`
	Categories []importMaterialCategory `json:"new_categories"`
}
type importMaterialRow struct {
	Row      uint32 `json:"row"`
	Name     string `json:"name_hex"`
	Key      string `json:"name_key_hex"`
	Phone    string `json:"phone_hex"`
	Category string `json:"category_key_hex"`
}
type importMaterialCategory struct {
	Name string `json:"name_hex"`
	Key  string `json:"name_key_hex"`
}

func commitMaterial(f importRPCResult) importMaterial {
	m := importMaterial{Status: "UNAVAILABLE", Rows: []importMaterialRow{}, Categories: []importMaterialCategory{}}
	if f.Plan == nil {
		return m
	}
	var plan previewPlan
	if json.Unmarshal([]byte(*f.Plan), &plan) != nil || plan.Kind != f.Kind {
		return m
	}
	for _, r := range plan.Rows {
		if r.Outcome == "ERROR" {
			continue
		}
		name, key := r.Name, catalogKey(r.Name)
		if f.Kind == Staff {
			name, key = staffName(r.Name)
		}
		m.Rows = append(m.Rows, importMaterialRow{Row: r.Row, Name: importHex(name), Key: hex.EncodeToString(key), Phone: importHex(r.Phone), Category: hex.EncodeToString(catalogKey(r.Category))})
	}
	if f.Kind != Staff {
		for _, name := range plan.NewCategories {
			m.Categories = append(m.Categories, importMaterialCategory{Name: importHex(name), Key: hex.EncodeToString(catalogKey(name))})
		}
	}
	m.Status = "VALID"
	return m
}
func (a *SharedPGApplication) Commit(ctx context.Context, meta WriteMeta, token string, skip bool) (CommitResult, error) {
	if a == nil || a.client == nil || ctx == nil {
		return CommitResult{}, ErrUnavailable
	}
	tokenHash := sha256.Sum256([]byte(token))
	request := struct {
		TokenHash [32]byte
		Skip      bool
	}{tokenHash, skip}
	p := importReceiptParameters(meta, request)
	p["p_token_hash"] = hex.EncodeToString(tokenHash[:])
	p["p_skip"] = skip
	p["p_plan_digest"] = nil
	p["p_material"] = nil
	p["p_observed_at"] = nil
	var f importRPCResult
	if a.client.CallRPC(ctx, "order_import_commit_v1", p, &f) != nil {
		return CommitResult{}, ErrUnavailable
	}
	if f.Outcome == "NEED_MATERIAL" {
		if f.Plan == nil || f.ObservedAt == nil || f.PlanDigest != importRawDigest(*f.Plan) {
			return CommitResult{}, ErrUnavailable
		}
		p["p_plan_digest"] = f.PlanDigest
		material := commitMaterial(f)
		p["p_material"] = material
		p["p_observed_at"] = *f.ObservedAt
		if material.Status == "VALID" {
			if a.receiptNow == nil {
				return CommitResult{}, ErrUnavailable
			}
			p["p_receipt_at"] = a.receiptNow().UTC().Truncate(time.Microsecond)
		}
		if a.client.CallRPC(ctx, "order_import_commit_v1", p, &f) != nil {
			return CommitResult{}, ErrUnavailable
		}
	}
	var out CommitResult
	switch f.Outcome {
	case "REPLAY":
		if err := importReplay(f, p["p_request_digest"].(string), &out); err != nil {
			return CommitResult{}, err
		}
		out.Duplicate = true
		return out, nil
	case "SUCCEEDED":
		if f.Response == nil || json.Unmarshal([]byte(*f.Response), &out) != nil {
			return CommitResult{}, ErrUnavailable
		}
		return out, nil
	default:
		return CommitResult{}, importOutcome(f.Outcome)
	}
}
func importRawDigest(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

var _ Application = (*SharedPGApplication)(nil)

func (a *SharedPGApplication) Result(ctx context.Context, actor, id uint64) (CommitResult, error) {
	var f importRPCResult
	if a == nil || a.client == nil || a.client.CallRPC(ctx, "order_import_result_v1", map[string]any{"p_user": strconv.FormatUint(actor, 10), "p_batch": strconv.FormatUint(id, 10)}, &f) != nil {
		return CommitResult{}, ErrUnavailable
	}
	var out CommitResult
	if f.Outcome != "SUCCEEDED" || f.Response == nil || json.Unmarshal([]byte(*f.Response), &out) != nil || out.Rows == nil {
		return CommitResult{}, ErrUnavailable
	}
	return out, nil
}
