package migrate

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"reflect"

	"github.com/gaofeng30/order/services/api/migrations/postgresql"
)

type sharedPostgreSQLCatalog struct {
	Ledger         []map[string]any `json:"ledger"`
	HistoryColumns []map[string]any `json:"history_columns"`
	HistoryPrimary bool             `json:"history_primary"`
	Columns        []map[string]any `json:"columns"`
	Constraints    []map[string]any `json:"constraints"`
	Sequences      []map[string]any `json:"sequences"`
}

// SharedPostgreSQLCatalogReadinessSQL returns one SELECT expression containing
// actual catalog facts. Only the embedded authoritative object names select the
// bounded catalog inventory; compatibility is determined by Go, not by SQL.
func SharedPostgreSQLCatalogReadinessSQL() (string, error) {
	want, err := sharedPostgreSQLCatalogAuthority()
	if err != nil {
		return "", err
	}
	columns, err := json.Marshal(want.Columns)
	if err != nil {
		return "", migrateError{reason: ReasonDatabaseIncompatible}
	}
	constraints, err := json.Marshal(want.Constraints)
	if err != nil {
		return "", migrateError{reason: ReasonDatabaseIncompatible}
	}
	sequences, err := json.Marshal(want.Sequences)
	if err != nil {
		return "", migrateError{reason: ReasonDatabaseIncompatible}
	}
	return fmt.Sprintf(`(SELECT pg_catalog.jsonb_build_object(
  'ledger', (SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'version',m.version,'name',m.name,'checksum',pg_catalog.encode(m.checksum,'hex'),'dirty',m.dirty
  ) ORDER BY m.version),'[]'::jsonb) FROM public.schema_migrations m),
  'history_columns', (SELECT COALESCE(pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'name',column_name,'data_type',data_type,'nullable',is_nullable,
    'length',COALESCE(character_maximum_length,0)
  ) ORDER BY ordinal_position),'[]'::jsonb) FROM information_schema.columns
    WHERE table_schema='public' AND table_name='schema_migrations'),
  'history_primary', EXISTS(SELECT 1 FROM pg_catalog.pg_constraint
    WHERE conrelid='public.schema_migrations'::regclass AND contype='p' AND conkey=ARRAY[1]::smallint[]),
  'columns', (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'table_name',e.table_name,'column_name',e.column_name,
    'present',COALESCE(a.attnum>0 AND NOT a.attisdropped,FALSE),
    'relation_kind',c.relkind::text,'data_type',pg_catalog.format_type(a.atttypid,a.atttypmod),
    'not_null',a.attnotnull,'column_default',pg_catalog.pg_get_expr(d.adbin,d.adrelid)
  ) ORDER BY e.table_name COLLATE "C",e.column_name COLLATE "C")
    FROM pg_catalog.jsonb_to_recordset(%s::jsonb) AS e(table_name text,column_name text)
    LEFT JOIN pg_catalog.pg_namespace n ON n.nspname='public'
    LEFT JOIN pg_catalog.pg_class c ON c.relnamespace=n.oid AND c.relname=e.table_name
    LEFT JOIN pg_catalog.pg_attribute a ON a.attrelid=c.oid AND a.attname=e.column_name
    LEFT JOIN pg_catalog.pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum),
  'constraints', (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'table_name',e.table_name,'constraint_name',e.constraint_name,'present',k.oid IS NOT NULL,
    'kind',k.contype::text,'definition',pg_catalog.pg_get_constraintdef(k.oid,true),
    'validated',k.convalidated,'deferrable',k.condeferrable,
    'index_valid',CASE WHEN k.contype IN ('p','u') THEN i.indisvalid ELSE NULL END,
    'index_ready',CASE WHEN k.contype IN ('p','u') THEN i.indisready ELSE NULL END
  ) ORDER BY e.table_name COLLATE "C",e.constraint_name COLLATE "C")
    FROM pg_catalog.jsonb_to_recordset(%s::jsonb) AS e(table_name text,constraint_name text)
    LEFT JOIN pg_catalog.pg_namespace n ON n.nspname='public'
    LEFT JOIN pg_catalog.pg_class c ON c.relnamespace=n.oid AND c.relname=e.table_name
    LEFT JOIN pg_catalog.pg_constraint k ON k.conrelid=c.oid AND k.conname=e.constraint_name
    LEFT JOIN pg_catalog.pg_index i ON i.indexrelid=k.conindid),
  'sequences', (SELECT pg_catalog.jsonb_agg(pg_catalog.jsonb_build_object(
    'sequence_name',e.sequence_name,'present',v.seqrelid IS NOT NULL,'relation_kind',s.relkind::text,
    'data_type',pg_catalog.format_type(v.seqtypid,NULL),'start_value',v.seqstart,
    'increment_by',v.seqincrement,'min_value',v.seqmin,'max_value',v.seqmax,
    'cache_value',v.seqcache,'cycle',v.seqcycle,
    'owner_schema',tn.nspname,'owner_table',t.relname,'owner_column',a.attname
  ) ORDER BY e.sequence_name COLLATE "C")
    FROM pg_catalog.jsonb_to_recordset(%s::jsonb) AS e(sequence_name text)
    LEFT JOIN pg_catalog.pg_namespace n ON n.nspname='public'
    LEFT JOIN pg_catalog.pg_class s ON s.relnamespace=n.oid AND s.relname=e.sequence_name
    LEFT JOIN pg_catalog.pg_sequence v ON v.seqrelid=s.oid
    LEFT JOIN pg_catalog.pg_depend d ON d.classid='pg_class'::regclass AND d.objid=s.oid AND d.deptype IN ('a','i')
    LEFT JOIN pg_catalog.pg_class t ON t.oid=d.refobjid
    LEFT JOIN pg_catalog.pg_namespace tn ON tn.oid=t.relnamespace
    LEFT JOIN pg_catalog.pg_attribute a ON a.attrelid=t.oid AND a.attnum=d.refobjsubid)
))`, sharedPostgreSQLLiteral(string(columns)), sharedPostgreSQLLiteral(string(constraints)), sharedPostgreSQLLiteral(string(sequences))), nil
}

// VerifySharedPostgreSQLCatalogReadiness verifies raw facts against the native
// immutable migration chain and final schema contract.
func VerifySharedPostgreSQLCatalogReadiness(raw json.RawMessage) error {
	want, err := sharedPostgreSQLCatalogAuthority()
	if err != nil {
		return err
	}
	expected, err := json.Marshal(want)
	if err != nil {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	actualFacts, err := sharedPostgreSQLCatalogJSON(raw)
	if err != nil {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	expectedFacts, err := sharedPostgreSQLCatalogJSON(expected)
	if err != nil || !reflect.DeepEqual(actualFacts, expectedFacts) {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	return nil
}

func sharedPostgreSQLCatalogJSON(raw []byte) (any, error) {
	decoder := json.NewDecoder(bytes.NewReader(raw))
	// Sequence limits and uint64 migration versions must never pass through a
	// float64 conversion that can silently round a changed catalog value.
	decoder.UseNumber()
	var result any
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return nil, migrateError{reason: ReasonDatabaseIncompatible}
	}
	return result, nil
}

func sharedPostgreSQLCatalogAuthority() (sharedPostgreSQLCatalog, error) {
	var want sharedPostgreSQLCatalog
	chain, err := Load(postgresql.FS)
	if err != nil {
		return want, err
	}
	var contract postgreSQLSchemaContract
	if json.Unmarshal(postgreSQLSchemaJSON, &contract) != nil || contract.Version != 45 || len(chain) != 45 || len(contract.Migrations) != 45 || len(contract.Tables) == 0 || len(contract.Sequences) == 0 {
		return want, migrateError{reason: ReasonDatabaseIncompatible}
	}
	for _, migration := range chain {
		checksum := hex.EncodeToString(migration.Checksum[:])
		if contract.Migrations["services/api/migrations/postgresql/"+migration.Name] != checksum || sha256.Sum256(migration.SQL) != migration.Checksum {
			return want, migrateError{reason: ReasonDatabaseIncompatible}
		}
		want.Ledger = append(want.Ledger, map[string]any{"version": migration.Version, "name": migration.Name, "checksum": checksum, "dirty": false})
	}
	want.HistoryPrimary = true
	want.HistoryColumns = []map[string]any{
		{"name": "version", "data_type": "bigint", "nullable": "NO", "length": 0},
		{"name": "name", "data_type": "character varying", "nullable": "NO", "length": 255},
		{"name": "checksum", "data_type": "bytea", "nullable": "NO", "length": 0},
		{"name": "dirty", "data_type": "boolean", "nullable": "NO", "length": 0},
		{"name": "applied_at", "data_type": "timestamp with time zone", "nullable": "YES", "length": 0},
	}
	for _, table := range sharedPostgreSQLKeys(contract.Tables) {
		for _, name := range sharedPostgreSQLKeys(contract.Tables[table].Columns) {
			column := contract.Tables[table].Columns[name]
			want.Columns = append(want.Columns, map[string]any{"table_name": table, "column_name": name, "present": true, "relation_kind": "r", "data_type": column.Type, "not_null": column.NotNull, "column_default": column.Default})
		}
		for _, name := range sharedPostgreSQLKeys(contract.Tables[table].Constraints) {
			constraint := contract.Tables[table].Constraints[name]
			var indexValid, indexReady any
			if constraint.Kind == "p" || constraint.Kind == "u" {
				indexValid, indexReady = true, true
			}
			want.Constraints = append(want.Constraints, map[string]any{"table_name": table, "constraint_name": name, "present": true, "kind": constraint.Kind, "definition": constraint.Definition, "validated": constraint.Validated, "deferrable": constraint.Deferrable, "index_valid": indexValid, "index_ready": indexReady})
		}
	}
	for _, name := range sharedPostgreSQLKeys(contract.Sequences) {
		sequence := contract.Sequences[name]
		want.Sequences = append(want.Sequences, map[string]any{"sequence_name": name, "present": true, "relation_kind": "S", "data_type": sequence.Type, "start_value": sequence.Start, "increment_by": sequence.Increment, "min_value": sequence.Min, "max_value": sequence.Max, "cache_value": sequence.Cache, "cycle": sequence.Cycle, "owner_schema": "public", "owner_table": sequence.OwnerTable, "owner_column": sequence.OwnerColumn})
	}
	return want, nil
}
