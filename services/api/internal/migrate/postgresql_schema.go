package migrate

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"sort"
)

// Generated from the authoritative PG migrations by pg-local-gate.py
// --capture-schema, then reviewed against their declared final objects. This
// bounded runtime contract ignores additional, unused objects and business rows.
// Regenerate and review when the migration chain or repository schema changes.
//
//go:embed postgresql_schema_contract.json
var postgreSQLSchemaJSON []byte

type postgreSQLSchemaContract struct {
	Version    uint64            `json:"version"`
	Migrations map[string]string `json:"migrations"`
	Sequences  map[string]struct {
		Type        string `json:"type"`
		Start       int64  `json:"start"`
		Increment   int64  `json:"increment"`
		Min         int64  `json:"min"`
		Max         int64  `json:"max"`
		Cache       int64  `json:"cache"`
		Cycle       bool   `json:"cycle"`
		OwnerTable  string `json:"owner_table"`
		OwnerColumn string `json:"owner_column"`
	} `json:"sequences"`
	Tables map[string]struct {
		Columns map[string]struct {
			Type    string  `json:"type"`
			NotNull bool    `json:"not_null"`
			Default *string `json:"default"`
		} `json:"columns"`
		Constraints map[string]struct {
			Kind       string `json:"kind"`
			Definition string `json:"definition"`
			Validated  bool   `json:"validated"`
			Deferrable bool   `json:"deferrable"`
		} `json:"constraints"`
	} `json:"tables"`
}

func verifyPostgreSQLBusinessSchema(ctx context.Context, q postgresQueryer, migrations []Migration) error {
	var contract postgreSQLSchemaContract
	if json.Unmarshal(postgreSQLSchemaJSON, &contract) != nil || len(contract.Tables) == 0 {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	// Historical-prefix fixtures have their own history contract. The complete
	// runtime schema is required once all versions in this manifest are present.
	if uint64(len(migrations)) < contract.Version {
		return nil
	}
	if uint64(len(migrations)) != contract.Version || len(contract.Migrations) != len(migrations) {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	for _, migration := range migrations {
		if contract.Migrations["services/api/migrations/postgresql/"+migration.Name] != hex.EncodeToString(migration.Checksum[:]) || sha256.Sum256(migration.SQL) != migration.Checksum {
			return migrateError{reason: ReasonDatabaseIncompatible}
		}
	}
	var columns, constraints, sequences []map[string]any
	names := make([]string, 0, len(contract.Tables))
	for name := range contract.Tables {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, table := range names {
		for name, column := range contract.Tables[table].Columns {
			columns = append(columns, map[string]any{"table_name": table, "column_name": name, "data_type": column.Type, "not_null": column.NotNull, "column_default": column.Default})
		}
		for name, constraint := range contract.Tables[table].Constraints {
			constraints = append(constraints, map[string]any{"table_name": table, "constraint_name": name, "kind": constraint.Kind, "definition": constraint.Definition, "validated": constraint.Validated, "deferrable": constraint.Deferrable})
		}
	}
	for name, sequence := range contract.Sequences {
		sequences = append(sequences, map[string]any{"sequence_name": name, "data_type": sequence.Type, "start_value": sequence.Start,
			"increment_by": sequence.Increment, "min_value": sequence.Min, "max_value": sequence.Max, "cache_value": sequence.Cache,
			"cycle": sequence.Cycle, "owner_table": sequence.OwnerTable, "owner_column": sequence.OwnerColumn})
	}
	columnJSON, err := json.Marshal(columns)
	if err != nil {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	constraintJSON, err := json.Marshal(constraints)
	if err != nil {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	sequenceJSON, err := json.Marshal(sequences)
	if err != nil || len(sequences) == 0 {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	var compatible bool
	// The aggregates evaluate a fixed number of manifest entries using catalog
	// joins. No application rows or unbounded whole-schema inventory is read.
	err = q.QueryRowContext(ctx, `SELECT COALESCE(bool_and(COALESCE(
		c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped
		AND format_type(a.atttypid,a.atttypmod)=e.data_type AND a.attnotnull=e.not_null
		AND pg_get_expr(d.adbin,d.adrelid) IS NOT DISTINCT FROM e.column_default,FALSE)),FALSE)
		FROM jsonb_to_recordset($1::jsonb) AS e(table_name text,column_name text,data_type text,not_null boolean,column_default text)
		LEFT JOIN pg_namespace n ON n.nspname='public'
		LEFT JOIN pg_class c ON c.relnamespace=n.oid AND c.relname=e.table_name
		LEFT JOIN pg_attribute a ON a.attrelid=c.oid AND a.attname=e.column_name
		LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum`, string(columnJSON)).Scan(&compatible)
	if err != nil {
		return migrateError{reason: ReasonDatabaseUnreachable}
	}
	if !compatible {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	err = q.QueryRowContext(ctx, `SELECT COALESCE(bool_and(COALESCE(
		k.contype::text=e.kind AND pg_get_constraintdef(k.oid,true)=e.definition
		AND k.convalidated=e.validated AND k.condeferrable=e."deferrable"
		AND (k.contype NOT IN ('p','u') OR (i.indisvalid AND i.indisready)),FALSE)),FALSE)
		FROM jsonb_to_recordset($1::jsonb) AS e(table_name text,constraint_name text,kind text,definition text,validated boolean,"deferrable" boolean)
		LEFT JOIN pg_namespace n ON n.nspname='public'
		LEFT JOIN pg_class c ON c.relnamespace=n.oid AND c.relname=e.table_name
		LEFT JOIN pg_constraint k ON k.conrelid=c.oid AND k.conname=e.constraint_name
		LEFT JOIN pg_index i ON i.indexrelid=k.conindid`, string(constraintJSON)).Scan(&compatible)
	if err != nil {
		return migrateError{reason: ReasonDatabaseUnreachable}
	}
	if !compatible {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	err = q.QueryRowContext(ctx, `SELECT COALESCE(bool_and(COALESCE(
        s.relkind='S' AND format_type(v.seqtypid,NULL)=e.data_type AND v.seqstart=e.start_value
        AND v.seqincrement=e.increment_by AND v.seqmin=e.min_value AND v.seqmax=e.max_value
        AND v.seqcache=e.cache_value AND v.seqcycle=e.cycle
        AND tn.nspname='public' AND t.relname=e.owner_table AND a.attname=e.owner_column,FALSE)),FALSE)
        FROM jsonb_to_recordset($1::jsonb) AS e(sequence_name text,data_type text,start_value bigint,increment_by bigint,min_value bigint,max_value bigint,cache_value bigint,cycle boolean,owner_table text,owner_column text)
        LEFT JOIN pg_namespace n ON n.nspname='public'
        LEFT JOIN pg_class s ON s.relnamespace=n.oid AND s.relname=e.sequence_name
        LEFT JOIN pg_sequence v ON v.seqrelid=s.oid
        LEFT JOIN pg_depend d ON d.classid='pg_class'::regclass AND d.objid=s.oid AND d.deptype IN ('a','i')
        LEFT JOIN pg_class t ON t.oid=d.refobjid
        LEFT JOIN pg_namespace tn ON tn.oid=t.relnamespace
        LEFT JOIN pg_attribute a ON a.attrelid=t.oid AND a.attnum=d.refobjsubid`, string(sequenceJSON)).Scan(&compatible)
	if err != nil {
		return migrateError{reason: ReasonDatabaseUnreachable}
	}
	if !compatible {
		return migrateError{reason: ReasonDatabaseIncompatible}
	}
	return nil
}
