package importbatch

import (
	"context"
	"fmt"
	"golang.org/x/text/unicode/norm"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type previewPlan struct {
	Kind           Kind      `json:"kind"`
	Rows           []planRow `json:"rows"`
	IgnoredColumns []string  `json:"ignored_columns,omitempty"`
	NewCategories  []string  `json:"new_categories,omitempty"`
	NewCount       uint32    `json:"new_count"`
	UpdateCount    uint32    `json:"update_count"`
	ErrorCount     uint32    `json:"error_count"`
}
type planRow struct {
	Images      []ProductImage `json:"images,omitempty"`
	PriceInput  string         `json:"price_input,omitempty"`
	MealInput   string         `json:"meal_input,omitempty"`
	Row         uint32         `json:"row"`
	Outcome     string         `json:"outcome"`
	Reason      string         `json:"reason,omitempty"`
	Name        string         `json:"name,omitempty"`
	Phone       string         `json:"phone,omitempty"`
	Category    string         `json:"category,omitempty"`
	Meal        string         `json:"meal,omitempty"`
	Description string         `json:"description,omitempty"`
	PriceCents  uint32         `json:"price_cents,omitempty"`
}

func planHeader(kind Kind, rows [][]string) (map[string]int, previewPlan, error) {
	header := map[string]int{}
	known := map[string]bool{}
	required := []string{}
	if kind == Product {
		required = []string{"菜品名称", "售价", "分类", "餐段可售"}
		for _, v := range append(required, "描述") {
			known[v] = true
		}
	} else if kind == Staff {
		required = []string{"姓名", "手机号"}
		for _, v := range required {
			known[v] = true
		}
	} else {
		return nil, previewPlan{}, ErrInvalidInput
	}
	plan := previewPlan{Kind: kind}
	for i, v := range rows[0] {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if known[v] {
			if _, dup := header[v]; dup {
				return nil, previewPlan{}, ErrInvalidTemplate
			}
			header[v] = i
		} else {
			plan.IgnoredColumns = append(plan.IgnoredColumns, v)
		}
	}
	for _, v := range required {
		if _, ok := header[v]; !ok {
			return nil, previewPlan{}, ErrInvalidTemplate
		}
	}
	return header, plan, nil
}

func planRows(kind Kind, rows [][]string, header map[string]int, plan previewPlan, existingProducts, existingCategories, existingStaff map[string]bool) (previewPlan, error) {
	seen := map[string]uint32{}
	newCats := map[string]bool{}
	for i, row := range rows[1:] {
		line := uint32(i + 2)
		get := func(name string) string {
			idx, ok := header[name]
			if !ok || idx >= len(row) {
				return ""
			}
			return strings.TrimSpace(row[idx])
		}
		item := planRow{Row: line, Outcome: "ADD"}
		if kind == Product {
			item.Name = strings.TrimSpace(norm.NFKC.String(get("菜品名称")))
			item.Category = strings.TrimSpace(norm.NFKC.String(get("分类")))
			item.Meal = map[string]string{"全天": "all", "午餐": "lunch", "晚餐": "dinner"}[get("餐段可售")]
			item.Description = get("描述")
			item.PriceInput = get("售价")
			item.MealInput = get("餐段可售")
			var priceErr error
			item.PriceCents, priceErr = parseCents(get("售价"))
			key := item.Name
			if item.Name == "" || item.Category == "" || item.Meal == "" || priceErr != nil {
				item.Outcome, item.Reason = "ERROR", "必填字段或售价格式不正确"
			} else if seen[key] > 0 {
				item.Outcome, item.Reason = "ERROR", fmt.Sprintf("菜品名称在第 %d 行重复", seen[key])
			} else if existingProducts[key] {
				item.Outcome, item.Reason = "ERROR", "菜品已存在，导入只新增"
			} else {
				seen[key] = line
				catKey := item.Category
				if !existingCategories[catKey] && !newCats[catKey] {
					newCats[catKey] = true
					plan.NewCategories = append(plan.NewCategories, item.Category)
				}
			}
		} else {
			item.Name = strings.TrimSpace(norm.NFKC.String(get("姓名")))
			item.Phone = canonicalImportPhone(get("手机号"))
			key := item.Phone
			if item.Name == "" || item.Phone == "" {
				item.Outcome, item.Reason = "ERROR", "姓名或手机号格式不正确"
			} else if seen[key] > 0 {
				item.Outcome, item.Reason = "ERROR", fmt.Sprintf("手机号在第 %d 行重复", seen[key])
			} else {
				seen[key] = line
				if existingStaff[key] {
					item.Outcome = "UPDATE"
				}
			}
		}
		switch item.Outcome {
		case "ADD":
			plan.NewCount++
		case "UPDATE":
			plan.UpdateCount++
		default:
			plan.ErrorCount++
		}
		plan.Rows = append(plan.Rows, item)
	}
	return plan, nil
}
func planPreview(token string, p previewPlan, expires time.Time) Preview {
	rows := make([]RowOutcome, 0, len(p.Rows))
	for _, r := range p.Rows {
		out := RowOutcome{Row: r.Row, Outcome: r.Outcome, Reason: r.Reason}
		if p.Kind == Product {
			product := r
			out.Product = &product
		}
		rows = append(rows, out)
	}
	return Preview{Token: token, NewCount: p.NewCount, UpdateCount: p.UpdateCount, ErrorCount: p.ErrorCount, NewCategories: p.NewCategories, IgnoredColumns: p.IgnoredColumns, Rows: rows, ExpiresAt: expires}
}
func parseCents(v string) (uint32, error) {
	parts := strings.Split(v, ".")
	if len(parts) > 2 || parts[0] == "" {
		return 0, ErrInvalidInput
	}
	whole, err := strconv.ParseUint(parts[0], 10, 32)
	if err != nil {
		return 0, err
	}
	frac := uint64(0)
	if len(parts) == 2 {
		if len(parts[1]) == 1 {
			parts[1] += "0"
		}
		if len(parts[1]) != 2 {
			return 0, ErrInvalidInput
		}
		frac, err = strconv.ParseUint(parts[1], 10, 8)
		if err != nil {
			return 0, err
		}
	}
	total := whole*100 + frac
	if total == 0 || total > uint64(^uint32(0)) {
		return 0, ErrInvalidInput
	}
	return uint32(total), nil
}
func canonicalImportPhone(v string) string {
	v = strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(v), " ", ""), "-", "")
	if len(v) == 11 && v[0] == '1' {
		return "+86" + v
	}
	if len(v) >= 2 && len(v) <= 16 && v[0] == '+' {
		for _, r := range v[1:] {
			if r < '0' || r > '9' {
				return ""
			}
		}
		return v
	}
	return ""
}
func catalogKey(v string) []byte { return []byte(strings.TrimSpace(norm.NFKC.String(v))) }
func staffName(v string) (string, []byte) {
	name := strings.TrimSpace(norm.NFKC.String(v))
	key := strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, name)
	return name, []byte(key)
}

// Images are confirmed uploads; drafts only persist references, never create products.
func attachImages(ctx context.Context, plan *previewPlan, images map[uint32][]ProductImage, reader ImageConfirmationReader) error {
	keys := []string{}
	for row, items := range images {
		if plan.Kind != Product || row < 2 || int(row) > len(plan.Rows)+1 || len(items) > 3 {
			return ErrInvalidInput
		}
		seen := map[string]bool{}
		for _, item := range items {
			if item.ObjectKey == "" || len(item.ObjectKey) > 1024 || seen[item.ObjectKey] {
				return ErrInvalidInput
			}
			seen[item.ObjectKey] = true
			keys = append(keys, item.ObjectKey)
		}
		plan.Rows[row-2].Images = items
	}
	if len(keys) == 0 {
		return nil
	}
	if reader == nil {
		return ErrUnavailable
	}
	confirmed, err := reader.Confirmed(ctx, keys)
	if err != nil {
		return ErrUnavailable
	}
	for _, key := range keys {
		if !confirmed[key] {
			return ErrInvalidInput
		}
	}
	return nil
}
