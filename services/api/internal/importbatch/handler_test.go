package importbatch

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"
)

func xlsxFixture(t *testing.T, rows string) []byte {
	t.Helper()
	var b bytes.Buffer
	w := zip.NewWriter(&b)
	s, err := w.Create("xl/worksheets/sheet1.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = s.Write([]byte(`<worksheet><sheetData>` + rows + `</sheetData></worksheet>`))
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}
func TestParseRowsReadsFirstWorksheetWithoutClientXLSXTruth(t *testing.T) {
	data := xlsxFixture(t, `<row><c r="A1" t="inlineStr"><is><t>姓名</t></is></c><c r="B1" t="inlineStr"><is><t>手机号</t></is></c></row><row><c r="A2" t="inlineStr"><is><t>张三</t></is></c><c r="B2"><v>13800138000</v></c></row>`)
	rows, err := ParseRows(data, 5000)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[1][0] != "张三" || rows[1][1] != "13800138000" {
		t.Fatalf("rows=%#v", rows)
	}
}
func TestParseRowsFailsClosedOverLimit(t *testing.T) {
	data := xlsxFixture(t, `<row><c r="A1"><v>header</v></c></row><row><c r="A2"><v>1</v></c></row><row><c r="A3"><v>2</v></c></row>`)
	_, err := ParseRows(data, 1)
	if !errors.Is(err, ErrTooManyRows) {
		t.Fatalf("err=%v", err)
	}
}
