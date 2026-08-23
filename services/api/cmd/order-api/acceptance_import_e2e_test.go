package main

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/gaofeng30/order/services/api/internal/httpapi"
	"github.com/gaofeng30/order/services/api/internal/identity"
	"github.com/gaofeng30/order/services/api/internal/importbatch"
	"github.com/gaofeng30/order/services/api/internal/merchantidentity"
)

// TestAcceptanceImportBoundariesAreDurable crosses the production router,
// OWNER PC authentication, multipart handlers and a fresh v1-v44 MySQL schema.
// It keeps rejected files at zero writes and proves accepted preview/commit
// facts through their public HTTP responses plus durable business/audit rows.
func TestAcceptanceImportBoundariesAreDurable(t *testing.T) {
	db := acceptanceFreshMySQL(t)
	acceptanceSeedSharedFacts(t, db)

	loginProvider := acceptanceLoginProvider{openIDs: map[string]string{
		"import-owner-login": "acceptance-import-owner-openid",
	}}
	phoneProvider := acceptancePhoneProvider{phones: map[string]string{
		"acceptance-import-owner-openid": acceptanceOwnerPhone,
	}}
	sessions := identity.NewService(loginProvider, identity.NewRepository(db))
	phoneService := identity.NewPhoneService(phoneProvider, identity.NewRepository(db))
	merchantRepository := merchantidentity.NewRepository(db)
	merchantService := merchantidentity.NewService(merchantRepository, phoneProvider)
	merchantAdmin := merchantidentity.NewMySQLAdminApplication(db, merchantService)
	merchantHandler := merchantidentity.NewAdminHandler(merchantAdmin)
	imports := importbatch.NewHandler(importbatch.NewMySQLApplication(db))

	router := httpapi.NewRouter(
		slog.New(slog.NewTextHandler(io.Discard, nil)),
		func(context.Context) httpapi.ReadinessResult { return httpapi.ReadinessResult{Ready: true} },
		identity.NewHandler(sessions),
		identity.NewPhoneHandler(sessions, phoneService),
		merchantidentity.NewHandler(sessions, merchantService),
		newAdminRoutes(sessions, merchantAdmin, merchantHandler, []adminGroupRegistrar{imports}, nil, imports),
	)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	client := server.Client()

	ownerToken := acceptanceMiniSession(t, client, server.URL, "import-owner-login")
	acceptanceBindPhone(t, client, server.URL, ownerToken, "import-owner-phone")
	acceptanceMerchantLogin(t, client, server.URL, ownerToken, "OWNER")
	pcToken := acceptanceImportPCSession(t, client, server.URL, ownerToken)

	productPreviewURL := server.URL + "/api/v1/admin/products/import/preview"
	staffPreviewURL := server.URL + "/api/v1/admin/staff-whitelist/import/preview"
	baseline := acceptanceImportCountsNow(t, db)

	// BE-27: the server rejects a non-xlsx multipart upload without a batch,
	// business row or audit receipt.
	acceptanceImportExpectError(t, acceptanceImportMultipart(t, client, productPreviewURL, pcToken, "import-non-xlsx", "menu.csv", []byte("not xlsx"), http.StatusUnprocessableEntity), "INVALID_FILE")
	acceptanceImportAssertCounts(t, db, baseline, "non-xlsx")

	// BE-28: a syntactically valid workbook missing a required product header
	// is a file-level template error and creates no preview batch.
	missingHeader := acceptanceImportXLSX(t, [][]string{
		{"菜品名称", "售价", "分类"},
		{"缺餐段菜品", "12.00", "新分类"},
	})
	acceptanceImportExpectError(t, acceptanceImportMultipart(t, client, productPreviewURL, pcToken, "import-missing-header", "missing-header.xlsx", missingHeader, http.StatusUnprocessableEntity), "INVALID_TEMPLATE")
	acceptanceImportAssertCounts(t, db, baseline, "missing required header")

	// BE-29: enforce all three independent server limits before persistence.
	tooLarge := make([]byte, 10*1024*1024+1)
	acceptanceImportExpectError(t, acceptanceImportMultipart(t, client, productPreviewURL, pcToken, "import-too-large", "too-large.xlsx", tooLarge, http.StatusRequestEntityTooLarge), "FILE_TOO_LARGE")
	acceptanceImportAssertCounts(t, db, baseline, "file over 10 MiB")

	productRows := [][]string{{"菜品名称", "售价", "分类", "餐段可售"}}
	for index := 0; index < 501; index++ {
		productRows = append(productRows, []string{fmt.Sprintf("超限菜品%03d", index), "1.00", "超限分类", "午餐"})
	}
	acceptanceImportExpectError(t, acceptanceImportMultipart(t, client, productPreviewURL, pcToken, "import-products-over-limit", "products-501.xlsx", acceptanceImportXLSX(t, productRows), http.StatusUnprocessableEntity), "TOO_MANY_ROWS")
	acceptanceImportAssertCounts(t, db, baseline, "product rows over 500")

	staffRows := [][]string{{"姓名", "手机号"}}
	for index := 0; index < 5001; index++ {
		staffRows = append(staffRows, []string{fmt.Sprintf("超限员工%04d", index), fmt.Sprintf("139%08d", index)})
	}
	acceptanceImportExpectError(t, acceptanceImportMultipart(t, client, staffPreviewURL, pcToken, "import-staff-over-limit", "staff-5001.xlsx", acceptanceImportXLSX(t, staffRows), http.StatusUnprocessableEntity), "TOO_MANY_ROWS")
	acceptanceImportAssertCounts(t, db, baseline, "staff rows over 5000")

	// BE-30: an existing product is an isolated row error. Committing with
	// skip_invalid records the batch but never overwrites the existing row.
	existingPreview := acceptanceImportMultipart(t, client, productPreviewURL, pcToken, "import-existing-preview", "existing-product.xlsx", acceptanceImportXLSX(t, [][]string{
		{"菜品名称", "售价", "分类", "餐段可售", "描述"},
		{"工作餐", "99.99", "验收套餐", "晚餐", "不得覆盖"},
	}), http.StatusCreated)
	acceptanceImportAssertPreview(t, existingPreview, 0, 0, 1)
	existingRows := acceptanceImportRows(t, existingPreview)
	if acceptanceString(t, existingRows[0], "outcome") != "ERROR" || !strings.Contains(acceptanceString(t, existingRows[0], "reason"), "已存在") {
		t.Fatal("existing product was not isolated as a row error")
	}
	existingCommit := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/import/commit", pcToken, "import-existing-commit", map[string]any{
		"preview_token": acceptanceString(t, existingPreview, "preview_token"), "skip_invalid": true,
	}, http.StatusOK)
	existingBatchID := acceptanceImportID(t, existingCommit, "batch_id")
	if acceptanceInt(t, existingCommit, "new_count") != 0 || acceptanceInt(t, existingCommit, "skipped_count") != 1 || acceptanceBool(t, existingCommit, "duplicate") {
		t.Fatal("existing product commit result did not preserve the isolated error")
	}
	acceptanceImportAssertOriginalProduct(t, db)

	// BE-31 and BE-33: one absent category is created once at the end and both
	// products use it. The same preview token/key replays the exact batch;
	// another key conflicts without a second business row or audit receipt.
	const importedCategory = "验收导入新分类"
	newPreview := acceptanceImportMultipart(t, client, productPreviewURL, pcToken, "import-new-category-preview", "new-category.xlsx", acceptanceImportXLSX(t, [][]string{
		{"菜品名称", "售价", "分类", "餐段可售", "描述"},
		{"验收导入菜品甲", "18.88", importedCategory, "午餐", "甲"},
		{"验收导入菜品乙", "9.90", importedCategory, "晚餐", "乙"},
	}), http.StatusCreated)
	acceptanceImportAssertPreview(t, newPreview, 2, 0, 0)
	newCategories := acceptanceImportStrings(t, newPreview, "new_categories")
	if len(newCategories) != 1 || newCategories[0] != importedCategory {
		t.Fatalf("new category preview = %#v", newCategories)
	}
	const newCommitKey = "import-new-category-commit"
	newCommit := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/import/commit", pcToken, newCommitKey, map[string]any{
		"preview_token": acceptanceString(t, newPreview, "preview_token"), "skip_invalid": false,
	}, http.StatusOK)
	newBatchID := acceptanceImportID(t, newCommit, "batch_id")
	if acceptanceInt(t, newCommit, "new_count") != 2 || acceptanceBool(t, newCommit, "duplicate") {
		t.Fatal("new category commit did not create the two accepted rows once")
	}
	replayed := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/import/commit", pcToken, newCommitKey, map[string]any{
		"preview_token": acceptanceString(t, newPreview, "preview_token"), "skip_invalid": false,
	}, http.StatusOK)
	if !acceptanceBool(t, replayed, "duplicate") || acceptanceImportID(t, replayed, "batch_id") != newBatchID || acceptanceInt(t, replayed, "new_count") != 2 {
		t.Fatal("same preview token and operation key did not replay the exact commit")
	}
	conflict := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/import/commit", pcToken, "import-new-category-conflict", map[string]any{
		"preview_token": acceptanceString(t, newPreview, "preview_token"), "skip_invalid": false,
	}, http.StatusConflict)
	acceptanceImportExpectError(t, conflict, "IDEMPOTENCY_CONFLICT")

	// BE-32: duplicate staff phone rows never use the last row. Only the first
	// valid row is committed and the conflicting name remains absent.
	const importedStaffPhone = "+8616600000095"
	staffPreview := acceptanceImportMultipart(t, client, staffPreviewURL, pcToken, "import-staff-duplicate-preview", "staff-duplicate.xlsx", acceptanceImportXLSX(t, [][]string{
		{"姓名", "手机号"},
		{"验收导入员工首行", importedStaffPhone},
		{"验收导入员工末行", importedStaffPhone},
	}), http.StatusCreated)
	acceptanceImportAssertPreview(t, staffPreview, 1, 0, 1)
	duplicateRows := acceptanceImportRows(t, staffPreview)
	if acceptanceString(t, duplicateRows[0], "outcome") != "ADD" || acceptanceString(t, duplicateRows[1], "outcome") != "ERROR" || !strings.Contains(acceptanceString(t, duplicateRows[1], "reason"), "重复") {
		t.Fatal("duplicate phone preview did not keep the first row and isolate the last")
	}
	staffCommit := acceptanceHTTP(t, client, http.MethodPost, server.URL+"/api/v1/import/commit", pcToken, "import-staff-duplicate-commit", map[string]any{
		"preview_token": acceptanceString(t, staffPreview, "preview_token"), "skip_invalid": true,
	}, http.StatusOK)
	staffBatchID := acceptanceImportID(t, staffCommit, "batch_id")
	if acceptanceInt(t, staffCommit, "new_count") != 1 || acceptanceInt(t, staffCommit, "skipped_count") != 1 || acceptanceBool(t, staffCommit, "duplicate") {
		t.Fatal("duplicate phone commit result was not one accepted and one skipped")
	}

	acceptanceImportAssertDurableFacts(t, db, existingBatchID, newBatchID, staffBatchID, importedCategory, importedStaffPhone)
}

func acceptanceImportPCSession(t *testing.T, client *http.Client, origin, ownerToken string) string {
	t.Helper()
	login := acceptanceQRLogin(t, acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/admin/auth/qrcode", "", "", map[string]any{}, http.StatusCreated))
	acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/me/admin-login/approve", ownerToken, "", map[string]any{
		"login_id": login.loginID, "approval_secret": login.approvalSecret, "code": "import-owner-approval",
	}, http.StatusOK)
	poll := acceptanceHTTP(t, client, http.MethodPost, origin+"/api/v1/admin/auth/poll", "", "", map[string]any{
		"login_id": login.loginID, "poll_secret": login.pollSecret,
	}, http.StatusOK)
	if acceptanceString(t, poll, "state") != "APPROVED" {
		t.Fatal("OWNER PC import session was not approved")
	}
	return acceptanceString(t, acceptanceObject(t, poll, "session"), "token")
}

func acceptanceImportMultipart(t *testing.T, client *http.Client, target, bearer, idempotency, filename string, data []byte, wantStatus int) map[string]any {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatal("create import multipart file")
	}
	if _, err := part.Write(data); err != nil {
		t.Fatal("write import multipart file")
	}
	if err := writer.Close(); err != nil {
		t.Fatal("close import multipart body")
	}
	request, err := http.NewRequestWithContext(t.Context(), http.MethodPost, target, bytes.NewReader(body.Bytes()))
	if err != nil {
		t.Fatal("build import multipart request")
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+bearer)
	request.Header.Set("Idempotency-Key", idempotency)
	response, err := client.Do(request)
	if err != nil {
		t.Fatalf("import multipart request failed: %v", err)
	}
	defer response.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(response.Body, 256*1024))
	if err != nil {
		t.Fatal("read import multipart response")
	}
	if response.StatusCode != wantStatus {
		t.Fatalf("import multipart %s status=%d want=%d body=%s", filename, response.StatusCode, wantStatus, raw)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var decoded map[string]any
	if decoder.Decode(&decoded) != nil {
		t.Fatal("decode import multipart response")
	}
	return decoded
}

func acceptanceImportXLSX(t *testing.T, rows [][]string) []byte {
	t.Helper()
	var sheet strings.Builder
	sheet.WriteString(`<worksheet><sheetData>`)
	for rowIndex, row := range rows {
		sheet.WriteString(`<row r="` + strconv.Itoa(rowIndex+1) + `">`)
		for columnIndex, value := range row {
			reference := acceptanceImportColumn(columnIndex+1) + strconv.Itoa(rowIndex+1)
			sheet.WriteString(`<c r="` + reference + `" t="inlineStr"><is><t>`)
			if err := xml.EscapeText(&sheet, []byte(value)); err != nil {
				t.Fatal("escape import xlsx cell")
			}
			sheet.WriteString(`</t></is></c>`)
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)

	var result bytes.Buffer
	archive := zip.NewWriter(&result)
	file, err := archive.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatal("create import xlsx worksheet")
	}
	if _, err := io.WriteString(file, sheet.String()); err != nil {
		t.Fatal("write import xlsx worksheet")
	}
	if err := archive.Close(); err != nil {
		t.Fatal("close import xlsx")
	}
	return result.Bytes()
}

func acceptanceImportColumn(value int) string {
	name := ""
	for value > 0 {
		remainder := (value - 1) % 26
		name = string(rune('A'+remainder)) + name
		value = (value - remainder - 1) / 26
	}
	return name
}

type acceptanceImportCounts struct {
	batches, categories, products, staff, audits int
}

func acceptanceImportCountsNow(t *testing.T, db *sql.DB) acceptanceImportCounts {
	t.Helper()
	var out acceptanceImportCounts
	err := db.QueryRowContext(t.Context(), `SELECT (SELECT COUNT(*) FROM import_batches),(SELECT COUNT(*) FROM categories),(SELECT COUNT(*) FROM products),(SELECT COUNT(*) FROM staff_whitelist),(SELECT COUNT(*) FROM action_audits)`).
		Scan(&out.batches, &out.categories, &out.products, &out.staff, &out.audits)
	if err != nil {
		t.Fatal("read import durability counts")
	}
	return out
}

func acceptanceImportAssertCounts(t *testing.T, db *sql.DB, want acceptanceImportCounts, label string) {
	t.Helper()
	if got := acceptanceImportCountsNow(t, db); got != want {
		t.Fatalf("%s wrote import facts: got=%#v want=%#v", label, got, want)
	}
}

func acceptanceImportExpectError(t *testing.T, response map[string]any, code string) {
	t.Helper()
	if acceptanceString(t, acceptanceObject(t, response, "error"), "code") != code {
		t.Fatalf("import error code did not equal %s", code)
	}
}

func acceptanceImportAssertPreview(t *testing.T, response map[string]any, newCount, updateCount, errorCount int64) {
	t.Helper()
	if acceptanceString(t, response, "preview_token") == "" ||
		acceptanceInt(t, response, "new_count") != newCount ||
		acceptanceInt(t, response, "update_count") != updateCount ||
		acceptanceInt(t, response, "error_count") != errorCount {
		t.Fatalf("unexpected import preview counts: %#v", response)
	}
}

func acceptanceImportRows(t *testing.T, response map[string]any) []map[string]any {
	t.Helper()
	raw, ok := response["rows"].([]any)
	if !ok || len(raw) == 0 {
		t.Fatal("import preview rows are missing")
	}
	rows := make([]map[string]any, len(raw))
	for index, value := range raw {
		var itemOK bool
		rows[index], itemOK = value.(map[string]any)
		if !itemOK {
			t.Fatalf("import preview row %d is malformed", index)
		}
	}
	return rows
}

func acceptanceImportStrings(t *testing.T, response map[string]any, key string) []string {
	t.Helper()
	raw, ok := response[key].([]any)
	if !ok {
		t.Fatalf("import response field %s is not an array", key)
	}
	values := make([]string, len(raw))
	for index, value := range raw {
		var itemOK bool
		values[index], itemOK = value.(string)
		if !itemOK {
			t.Fatalf("import response field %s[%d] is not a string", key, index)
		}
	}
	return values
}

func acceptanceImportID(t *testing.T, response map[string]any, key string) uint64 {
	t.Helper()
	id, err := strconv.ParseUint(acceptanceString(t, response, key), 10, 64)
	if err != nil || id == 0 {
		t.Fatalf("import response field %s is not an ID", key)
	}
	return id
}

func acceptanceImportAssertOriginalProduct(t *testing.T, db *sql.DB) {
	t.Helper()
	var count, categoryID, price int
	var description, meal string
	err := db.QueryRowContext(t.Context(), `SELECT COUNT(*),MIN(category_id),MIN(price_cents),MIN(description),MIN(meal_period) FROM products WHERE CONVERT(name_key USING utf8mb4)='工作餐'`).
		Scan(&count, &categoryID, &price, &description, &meal)
	if err != nil || count != 1 || categoryID != 1 || price != 1250 || description != "两荤一素" || meal != "lunch" {
		t.Fatalf("existing product was overwritten count=%d category=%d price=%d description=%q meal=%q err=%v", count, categoryID, price, description, meal, err)
	}
}

func acceptanceImportAssertDurableFacts(t *testing.T, db *sql.DB, existingBatchID, newBatchID, staffBatchID uint64, categoryName, staffPhone string) {
	t.Helper()
	acceptanceImportAssertOriginalProduct(t, db)

	var batches, committed, versioned, ownerBatches, productBatches, staffBatches int
	err := db.QueryRowContext(t.Context(), `SELECT COUNT(*),SUM(state='COMMITTED'),SUM(record_version=2),SUM(actor_account_id=1),SUM(kind='PRODUCT'),SUM(kind='STAFF') FROM import_batches`).
		Scan(&batches, &committed, &versioned, &ownerBatches, &productBatches, &staffBatches)
	if err != nil || batches != 3 || committed != 3 || versioned != 3 || ownerBatches != 3 || productBatches != 2 || staffBatches != 1 {
		t.Fatalf("import batches are not closed batches=%d committed=%d versioned=%d owner=%d products=%d staff=%d err=%v", batches, committed, versioned, ownerBatches, productBatches, staffBatches, err)
	}

	acceptanceImportAssertBatch(t, db, existingBatchID, "PRODUCT", 1, 0, 0, 1, true)
	acceptanceImportAssertBatch(t, db, newBatchID, "PRODUCT", 2, 2, 0, 0, false)
	acceptanceImportAssertBatch(t, db, staffBatchID, "STAFF", 2, 1, 0, 1, true)

	var categoryID uint64
	var categoryCount, categoryOrder int
	var categoryActive bool
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*),MIN(id),MIN(sort_order),MIN(is_active) FROM categories WHERE CONVERT(name_key USING utf8mb4)=?`, categoryName).
		Scan(&categoryCount, &categoryID, &categoryOrder, &categoryActive)
	if err != nil || categoryCount != 1 || categoryID == 0 || categoryOrder != 2 || !categoryActive {
		t.Fatalf("new category invariant count=%d id=%d order=%d active=%v err=%v", categoryCount, categoryID, categoryOrder, categoryActive, err)
	}

	var importedProducts, linkedProducts, listedProducts, exactPrices int
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*),SUM(category_id=?),SUM(is_listed=TRUE),SUM((name='验收导入菜品甲' AND price_cents=1888 AND meal_period='lunch') OR (name='验收导入菜品乙' AND price_cents=990 AND meal_period='dinner')) FROM products WHERE name IN ('验收导入菜品甲','验收导入菜品乙')`, categoryID).
		Scan(&importedProducts, &linkedProducts, &listedProducts, &exactPrices)
	if err != nil || importedProducts != 2 || linkedProducts != 2 || listedProducts != 2 || exactPrices != 2 {
		t.Fatalf("imported products diverged count=%d linked=%d listed=%d exact=%d err=%v", importedProducts, linkedProducts, listedProducts, exactPrices, err)
	}

	var staffCount int
	var staffName string
	var staffEnabled bool
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*),MIN(name),MIN(enabled) FROM staff_whitelist WHERE phone=?`, staffPhone).
		Scan(&staffCount, &staffName, &staffEnabled)
	if err != nil || staffCount != 1 || staffName != "验收导入员工首行" || !staffEnabled {
		t.Fatalf("duplicate staff phone used the wrong row count=%d name=%q enabled=%v err=%v", staffCount, staffName, staffEnabled, err)
	}

	var receipts, previewReceipts, commitReceipts, targets, validReceipts int
	err = db.QueryRowContext(t.Context(), `SELECT COUNT(*),SUM(action='PREVIEW_IMPORT'),SUM(action='COMMIT_IMPORT'),COUNT(DISTINCT target_id),SUM(entry_kind='COMMAND_RECEIPT' AND actor_kind='MERCHANT' AND actor_account_id=1 AND result='SUCCEEDED' AND reason_code='OK' AND target_type='IMPORT_BATCH') FROM action_audits WHERE action IN ('PREVIEW_IMPORT','COMMIT_IMPORT')`).
		Scan(&receipts, &previewReceipts, &commitReceipts, &targets, &validReceipts)
	if err != nil || receipts != 6 || previewReceipts != 3 || commitReceipts != 3 || targets != 3 || validReceipts != 6 {
		t.Fatalf("import audit receipts diverged count=%d preview=%d commit=%d targets=%d valid=%d err=%v", receipts, previewReceipts, commitReceipts, targets, validReceipts, err)
	}
}

func acceptanceImportAssertBatch(t *testing.T, db *sql.DB, batchID uint64, kind string, rowCount, newCount, updateCount, errorCount int, skipInvalid bool) {
	t.Helper()
	var gotKind, state string
	var gotRows, gotNew, gotUpdate, gotError int
	var gotSkip bool
	err := db.QueryRowContext(t.Context(), `SELECT kind,state,row_count,new_count,update_count,error_count,skip_invalid FROM import_batches WHERE id=?`, batchID).
		Scan(&gotKind, &state, &gotRows, &gotNew, &gotUpdate, &gotError, &gotSkip)
	if err != nil || gotKind != kind || state != "COMMITTED" || gotRows != rowCount || gotNew != newCount || gotUpdate != updateCount || gotError != errorCount || gotSkip != skipInvalid {
		t.Fatalf("batch %d kind=%s state=%s rows=%d new=%d update=%d error=%d skip=%v err=%v", batchID, gotKind, state, gotRows, gotNew, gotUpdate, gotError, gotSkip, err)
	}
}
