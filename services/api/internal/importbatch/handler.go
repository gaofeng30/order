package importbatch

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	ErrInvalidFile         = errors.New("invalid import file")
	ErrInvalidTemplate     = errors.New("invalid import template")
	ErrFileTooLarge        = errors.New("import file too large")
	ErrTooManyRows         = errors.New("too many import rows")
	ErrPreviewExpired      = errors.New("import preview expired")
	ErrInvalidInput        = errors.New("invalid import input")
	ErrIdempotencyConflict = errors.New("import idempotency conflict")
	ErrUnavailable         = errors.New("import unavailable")
)

type Kind string

const (
	Product     Kind = "PRODUCT"
	Staff       Kind = "STAFF"
	maxFileSize      = 10 * 1024 * 1024
)

func MaxRows(kind Kind) int {
	if kind == Staff {
		return 5000
	}
	return 500
}

type WriteMeta struct {
	ActorUserID               uint64
	IdempotencyKey, RequestID string
}
type XLSX struct {
	Name       string
	Bytes      []byte
	DigestHint string
}
type RowOutcome struct {
	Row             uint32
	Outcome, Reason string
}
type Preview struct {
	Token                             string
	NewCount, UpdateCount, ErrorCount uint32
	NewCategories, IgnoredColumns     []string
	Rows                              []RowOutcome
	ExpiresAt                         time.Time
}
type CommitResult struct {
	BatchID                             uint64
	NewCount, UpdateCount, SkippedCount uint32
	Duplicate                           bool
}

// Application persists one preview batch and atomically commits it with category/product or staff facts then audit receipt.
type Application interface {
	Preview(context.Context, WriteMeta, Kind, XLSX) (Preview, error)
	Commit(context.Context, WriteMeta, string, bool) (CommitResult, error)
}
type Handler struct{ app Application }

func NewHandler(app Application) *Handler { return &Handler{app: app} }
func (h *Handler) RegisterRoutes(group *gin.RouterGroup) {
	group.POST("/products/import/preview", h.preview(Product))
	group.POST("/staff-whitelist/import/preview", h.preview(Staff))
}

// RegisterCommitRoute mounts the frozen sibling /api/v1/import/commit path on the authenticated API group.
func (h *Handler) RegisterCommitRoute(api *gin.RouterGroup) { api.POST("/import/commit", h.commit) }
func (h *Handler) preview(kind Kind) gin.HandlerFunc {
	return func(c *gin.Context) {
		meta, ok := writeMeta(c)
		if !ok {
			writeErr(c, ErrInvalidInput)
			return
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxFileSize+4096)
		file, header, err := c.Request.FormFile("file")
		if err != nil {
			writeErr(c, ErrInvalidFile)
			return
		}
		defer file.Close()
		if strings.ToLower(filepath.Ext(header.Filename)) != ".xlsx" {
			writeErr(c, ErrInvalidFile)
			return
		}
		data, err := io.ReadAll(io.LimitReader(file, maxFileSize+1))
		if err != nil {
			writeErr(c, ErrInvalidFile)
			return
		}
		if len(data) > maxFileSize {
			writeErr(c, ErrFileTooLarge)
			return
		}
		out, err := h.app.Preview(c.Request.Context(), meta, kind, XLSX{Name: filepath.Base(header.Filename), Bytes: data})
		if err != nil {
			writeErr(c, err)
			return
		}
		c.JSON(http.StatusCreated, previewDTO(out))
	}
}

type commitWrite struct {
	PreviewToken string `json:"preview_token"`
	SkipInvalid  *bool  `json:"skip_invalid"`
}

func (h *Handler) commit(c *gin.Context) {
	meta, ok := writeMeta(c)
	var in commitWrite
	if !ok || !strictJSON(c, &in) || in.PreviewToken == "" || in.SkipInvalid == nil {
		writeErr(c, ErrInvalidInput)
		return
	}
	out, err := h.app.Commit(c.Request.Context(), meta, in.PreviewToken, *in.SkipInvalid)
	if err != nil {
		writeErr(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"batch_id": strconv.FormatUint(out.BatchID, 10), "new_count": out.NewCount, "update_count": out.UpdateCount, "skipped_count": out.SkippedCount, "duplicate": out.Duplicate})
}
func previewDTO(p Preview) gin.H {
	rows := make([]gin.H, 0, len(p.Rows))
	for _, r := range p.Rows {
		row := gin.H{"row": r.Row, "outcome": r.Outcome}
		if r.Reason != "" {
			row["reason"] = r.Reason
		}
		rows = append(rows, row)
	}
	return gin.H{"preview_token": p.Token, "new_count": p.NewCount, "update_count": p.UpdateCount, "error_count": p.ErrorCount, "new_categories": p.NewCategories, "ignored_columns": p.IgnoredColumns, "rows": rows, "expires_at": p.ExpiresAt}
}

// ParseRows is the bounded server-side XLSX seam used by Application implementations.
// It deliberately reads only the first worksheet; formulas/macros are not evaluated.
func ParseRows(data []byte, maxRows int) ([][]string, error) {
	if len(data) == 0 || len(data) > maxFileSize {
		return nil, ErrInvalidFile
	}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, ErrInvalidFile
	}
	files := map[string]*zip.File{}
	for _, f := range zr.File {
		files[f.Name] = f
	}
	shared := []string{}
	if f := files["xl/sharedStrings.xml"]; f != nil {
		shared, err = parseShared(f)
		if err != nil {
			return nil, ErrInvalidFile
		}
	}
	sheet := files["xl/worksheets/sheet1.xml"]
	if sheet == nil {
		return nil, ErrInvalidTemplate
	}
	rows, err := parseSheet(sheet, shared, maxRows+1)
	if errors.Is(err, ErrTooManyRows) {
		return nil, err
	}
	if err != nil || len(rows) == 0 {
		return nil, ErrInvalidTemplate
	}
	return rows, nil
}

type sharedStrings struct {
	Items []struct {
		Text string `xml:"t"`
		Runs []struct {
			Text string `xml:"t"`
		} `xml:"r"`
	} `xml:"si"`
}

func parseShared(file *zip.File) ([]string, error) {
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var doc sharedStrings
	if xml.NewDecoder(io.LimitReader(r, 4*1024*1024)).Decode(&doc) != nil {
		return nil, ErrInvalidFile
	}
	out := make([]string, 0, len(doc.Items))
	for _, item := range doc.Items {
		value := item.Text
		for _, run := range item.Runs {
			value += run.Text
		}
		out = append(out, value)
	}
	return out, nil
}

type sheetDoc struct {
	Rows []struct {
		Cells []struct {
			Ref    string `xml:"r,attr"`
			Type   string `xml:"t,attr"`
			Value  string `xml:"v"`
			Inline string `xml:"is>t"`
		} `xml:"c"`
	} `xml:"sheetData>row"`
}

func parseSheet(file *zip.File, shared []string, maxRows int) ([][]string, error) {
	r, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var doc sheetDoc
	if xml.NewDecoder(io.LimitReader(r, 16*1024*1024)).Decode(&doc) != nil {
		return nil, ErrInvalidFile
	}
	if len(doc.Rows) > maxRows {
		return nil, ErrTooManyRows
	}
	rows := make([][]string, 0, len(doc.Rows))
	for _, row := range doc.Rows {
		width := 0
		for _, cell := range row.Cells {
			col, ok := columnIndex(cell.Ref)
			if !ok {
				return nil, ErrInvalidTemplate
			}
			if col+1 > width {
				width = col + 1
			}
		}
		values := make([]string, width)
		for _, cell := range row.Cells {
			col, _ := columnIndex(cell.Ref)
			value := cell.Value
			if cell.Type == "s" {
				index, e := strconv.Atoi(value)
				if e != nil || index < 0 || index >= len(shared) {
					return nil, ErrInvalidTemplate
				}
				value = shared[index]
			} else if cell.Type == "inlineStr" {
				value = cell.Inline
			}
			values[col] = strings.TrimSpace(value)
		}
		rows = append(rows, values)
	}
	return rows, nil
}
func columnIndex(ref string) (int, bool) {
	letters := ""
	for _, r := range ref {
		if r >= 'A' && r <= 'Z' {
			letters += string(r)
		} else {
			break
		}
	}
	if letters == "" {
		return 0, false
	}
	n := 0
	for _, r := range letters {
		n = n*26 + int(r-'A'+1)
	}
	return n - 1, n > 0
}

func writeMeta(c *gin.Context) (WriteMeta, bool) {
	actor := c.GetUint64("actor_user_id")
	keys := c.Request.Header.Values("Idempotency-Key")
	if actor == 0 || len(keys) != 1 || strings.TrimSpace(keys[0]) == "" || strings.ContainsAny(keys[0], " \t\r\n") {
		return WriteMeta{}, false
	}
	return WriteMeta{actor, keys[0], c.GetString("request_id")}, true
}
func strictJSON(c *gin.Context, out any) bool {
	if !strings.HasPrefix(c.GetHeader("Content-Type"), "application/json") {
		return false
	}
	body, err := io.ReadAll(io.LimitReader(c.Request.Body, 16385))
	if err != nil || len(body) == 0 || len(body) > 16384 {
		return false
	}
	d := json.NewDecoder(bytes.NewReader(body))
	d.DisallowUnknownFields()
	if d.Decode(out) != nil {
		return false
	}
	var x any
	return errors.Is(d.Decode(&x), io.EOF)
}
func writeErr(c *gin.Context, err error) {
	status, code, message := http.StatusServiceUnavailable, "IMPORT_UNAVAILABLE", "import temporarily unavailable"
	switch {
	case errors.Is(err, ErrInvalidInput):
		status, code, message = http.StatusBadRequest, "INVALID_REQUEST", "invalid request"
	case errors.Is(err, ErrInvalidFile):
		status, code, message = http.StatusUnprocessableEntity, "INVALID_FILE", "invalid xlsx file"
	case errors.Is(err, ErrInvalidTemplate):
		status, code, message = http.StatusUnprocessableEntity, "INVALID_TEMPLATE", "invalid import template"
	case errors.Is(err, ErrFileTooLarge):
		status, code, message = http.StatusRequestEntityTooLarge, "FILE_TOO_LARGE", "file too large"
	case errors.Is(err, ErrTooManyRows):
		status, code, message = http.StatusUnprocessableEntity, "TOO_MANY_ROWS", "too many rows"
	case errors.Is(err, ErrPreviewExpired):
		status, code, message = http.StatusConflict, "PREVIEW_EXPIRED", "preview expired"
	case errors.Is(err, ErrIdempotencyConflict):
		status, code, message = http.StatusConflict, "IDEMPOTENCY_CONFLICT", "idempotency conflict"
	}
	c.JSON(status, gin.H{"error": gin.H{"code": code, "message": message}})
}
