package migrate

import (
	"crypto/sha256"
	"testing"
	"testing/fstest"
)

func TestLoadAcceptsContinuousSingleStatementMigrations(t *testing.T) {
	one := []byte("CREATE TABLE IF NOT EXISTS schema_migrations (version BIGINT);\n")
	two := []byte("CREATE TABLE probe (id BIGINT);\n")
	got, err := Load(fstest.MapFS{
		"000001_create_schema_migrations.sql": {Data: one},
		"000002_create_probe.sql":             {Data: two},
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(got) != 2 || got[0].Version != 1 || got[1].Version != 2 {
		t.Fatalf("migrations = %#v, want versions 1,2", got)
	}
	if got[0].Name != "000001_create_schema_migrations.sql" || got[0].Checksum != sha256.Sum256(one) {
		t.Fatalf("first migration metadata drifted: %#v", got[0])
	}
}

func TestLoadAcceptsSourceAsAColumnName(t *testing.T) {
	if _, err := Load(fstest.MapFS{
		"000001_create_observations.sql": {Data: []byte("CREATE TABLE observations (source ENUM('CALLBACK') NOT NULL);\n")},
	}); err != nil {
		t.Fatalf("Load() rejected source column: %v", err)
	}
}

func TestLoadRejectsInvalidMigrationSets(t *testing.T) {
	tests := map[string]fstest.MapFS{
		"empty":         {},
		"missing first": {"000002_second.sql": {Data: []byte("SELECT 1;\n")}},
		"duplicate version": {
			"000001_first.sql":  {Data: []byte("SELECT 1;\n")},
			"000001_second.sql": {Data: []byte("SELECT 2;\n")},
		},
		"gap": {
			"000001_first.sql": {Data: []byte("SELECT 1;\n")},
			"000003_third.sql": {Data: []byte("SELECT 3;\n")},
		},
		"invalid name":   {"1_bad.sql": {Data: []byte("SELECT 1;\n")}},
		"nested":         {"nested/000001_first.sql": {Data: []byte("SELECT 1;\n")}},
		"bom":            {"000001_first.sql": {Data: []byte("\xef\xbb\xbfSELECT 1;\n")}},
		"invalid utf8":   {"000001_first.sql": {Data: []byte{0xff, ';', '\n'}}},
		"crlf":           {"000001_first.sql": {Data: []byte("SELECT 1;\r\n")}},
		"no newline":     {"000001_first.sql": {Data: []byte("SELECT 1;")}},
		"two statements": {"000001_first.sql": {Data: []byte("SELECT 1; SELECT 2;\n")}},
		"delimiter":      {"000001_first.sql": {Data: []byte("DELIMITER //;\n")}},
		"source":         {"000001_first.sql": {Data: []byte("SOURCE secret.sql;\n")}},
		"load data":      {"000001_first.sql": {Data: []byte("LOAD DATA LOCAL INFILE 'x';\n")}},
		"down name":      {"000001_down.sql": {Data: []byte("SELECT 1;\n")}},
		"seed name":      {"000001_seed.sql": {Data: []byte("SELECT 1;\n")}},
	}

	for name, files := range tests {
		t.Run(name, func(t *testing.T) {
			if _, err := Load(files); err == nil {
				t.Fatal("Load() error = nil, want rejection")
			}
		})
	}
}
