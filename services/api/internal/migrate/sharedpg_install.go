package migrate

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gaofeng30/order/services/api/migrations/postgresql"
)

// SharedPostgreSQLInstallSQL generates a fixed management-plane installation.
// It accepts no SQL, migration source, object name or resume/version input. The
// original loader and final native schema contract remain authoritative.
// One DO statement installs all versions atomically, only into an empty public
// schema. This intentionally does not implement native per-version recovery.
func SharedPostgreSQLInstallSQL() (string, error) {
	chain, err := Load(postgresql.FS)
	if err != nil {
		return "", err
	}
	var contract postgreSQLSchemaContract
	if json.Unmarshal(postgreSQLSchemaJSON, &contract) != nil || contract.Version != 45 || len(chain) != 45 || len(contract.Migrations) != 45 || len(contract.Tables) == 0 || len(contract.Sequences) == 0 {
		return "", migrateError{reason: ReasonDatabaseIncompatible}
	}
	for _, migration := range chain {
		if contract.Migrations["services/api/migrations/postgresql/"+migration.Name] != hex.EncodeToString(migration.Checksum[:]) || sha256.Sum256(migration.SQL) != migration.Checksum {
			return "", migrateError{reason: ReasonDatabaseIncompatible}
		}
	}
	verification, err := sharedPostgreSQLVerification(chain, contract)
	if err != nil {
		return "", err
	}
	var body strings.Builder
	body.WriteString("DECLARE compatible boolean;\nBEGIN\n")
	fmt.Fprintf(&body, "  PERFORM pg_catalog.pg_advisory_xact_lock(%d::bigint);\n", postgresMigrationLockID)
	// Incoming statement bytes must already be UTF8; changing the setting cannot
	// repair text decoded using a different client encoding before DO executes.
	body.WriteString(`  IF current_setting('client_encoding') <> 'UTF8' OR current_setting('transaction_read_only') <> 'off'
     OR current_setting('server_version_num')::integer < 160000 OR current_setting('server_version_num')::integer >= 180000 THEN
    RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE='database_incompatible';
  END IF;
  PERFORM pg_catalog.set_config('TimeZone', 'UTC', true);
  PERFORM pg_catalog.set_config('search_path', 'public', true);
  PERFORM pg_catalog.set_config('standard_conforming_strings', 'on', true);
  IF current_setting('TimeZone') <> 'UTC' OR current_setting('client_encoding') <> 'UTF8'
     OR current_setting('search_path') <> 'public' OR current_setting('standard_conforming_strings') <> 'on'
     OR current_setting('transaction_read_only') <> 'off' THEN
    RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE='database_incompatible';
  END IF;
`)
	body.WriteString("  IF pg_catalog.to_regclass('public.schema_migrations') IS NOT NULL THEN\n")
	body.WriteString(verification)
	body.WriteString("    RETURN;\n  END IF;\n")
	body.WriteString(sharedPostgreSQLEmptySchemaCheck)
	for _, migration := range chain {
		// Dollar quoting preserves the exact bytes accepted by the original lexer,
		// including backslashes and the terminating newline. No scanner is relaxed.
		fmt.Fprintf(&body, "  EXECUTE %s;\n", sharedPostgreSQLLiteral(string(migration.SQL)))
		fmt.Fprintf(&body, "  INSERT INTO public.schema_migrations(version,name,checksum,dirty,applied_at) VALUES (%d,%s,pg_catalog.decode(%s,'hex'),FALSE,CURRENT_TIMESTAMP);\n", migration.Version, sharedPostgreSQLLiteral(migration.Name), sharedPostgreSQLLiteral(hex.EncodeToString(migration.Checksum[:])))
	}
	body.WriteString(verification)
	body.WriteString("END;\n")
	return "DO " + sharedPostgreSQLLiteral(body.String()) + ";\n", nil
}

// All namespace-local object categories are checked before any DDL or data
// write. The verified pgcrypto 1.3 platform dependency may already exist under
// the installation identity. Only its catalog-owned functions are excluded;
// similarly named objects, other extensions and other owners remain rejected.
const sharedPostgreSQLEmptySchemaCheck = `  IF EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE relnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_proc p WHERE pronamespace='public'::regnamespace AND NOT EXISTS (
       SELECT 1 FROM pg_catalog.pg_depend d JOIN pg_catalog.pg_extension e ON e.oid=d.refobjid
       WHERE d.classid='pg_catalog.pg_proc'::regclass AND d.objid=p.oid AND d.objsubid=0
         AND d.refclassid='pg_catalog.pg_extension'::regclass AND d.deptype='e'
         AND e.extname='pgcrypto' AND e.extversion='1.3' AND e.extnamespace='public'::regnamespace
         AND e.extowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname=current_user)))
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_type WHERE typnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_operator WHERE oprnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_collation WHERE collnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_conversion WHERE connamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_opclass WHERE opcnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_opfamily WHERE opfnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_ts_config WHERE cfgnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_ts_dict WHERE dictnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_ts_parser WHERE prsnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_ts_template WHERE tmplnamespace='public'::regnamespace)
     OR EXISTS (SELECT 1 FROM pg_catalog.pg_extension WHERE extnamespace='public'::regnamespace
       AND NOT (extname='pgcrypto' AND extversion='1.3' AND extowner=(SELECT oid FROM pg_catalog.pg_roles WHERE rolname=current_user))) THEN
    RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE='shared_install_public_not_empty';
  END IF;
`

func sharedPostgreSQLLiteral(value string) string {
	delimiter := "$order_data$"
	for index := 1; strings.Contains(value, delimiter); index++ {
		delimiter = "$order_data_" + strconv.Itoa(index) + "$"
	}
	return delimiter + value + delimiter
}

func sharedPostgreSQLKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sharedPostgreSQLVerification(chain []Migration, contract postgreSQLSchemaContract) (string, error) {
	var columns, constraints, sequences, history []map[string]any
	for _, migration := range chain {
		history = append(history, map[string]any{"version": migration.Version, "name": migration.Name, "checksum": hex.EncodeToString(migration.Checksum[:])})
	}
	for _, table := range sharedPostgreSQLKeys(contract.Tables) {
		for _, name := range sharedPostgreSQLKeys(contract.Tables[table].Columns) {
			column := contract.Tables[table].Columns[name]
			columns = append(columns, map[string]any{"table_name": table, "column_name": name, "data_type": column.Type, "not_null": column.NotNull, "column_default": column.Default})
		}
		for _, name := range sharedPostgreSQLKeys(contract.Tables[table].Constraints) {
			constraint := contract.Tables[table].Constraints[name]
			constraints = append(constraints, map[string]any{"table_name": table, "constraint_name": name, "kind": constraint.Kind, "definition": constraint.Definition, "validated": constraint.Validated, "deferrable": constraint.Deferrable})
		}
	}
	for _, name := range sharedPostgreSQLKeys(contract.Sequences) {
		sequence := contract.Sequences[name]
		sequences = append(sequences, map[string]any{"sequence_name": name, "data_type": sequence.Type, "start_value": sequence.Start,
			"increment_by": sequence.Increment, "min_value": sequence.Min, "max_value": sequence.Max, "cache_value": sequence.Cache,
			"cycle": sequence.Cycle, "owner_table": sequence.OwnerTable, "owner_column": sequence.OwnerColumn})
	}
	var result strings.Builder
	result.WriteString(sharedPostgreSQLHistoryShapeCheck)
	historyJSON, err := json.Marshal(history)
	if err != nil {
		return "", migrateError{reason: ReasonDatabaseIncompatible}
	}
	fmt.Fprintf(&result, `  IF EXISTS (SELECT 1 FROM public.schema_migrations WHERE dirty) THEN
    RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE='schema_dirty';
  END IF;
  IF (SELECT count(*) FROM public.schema_migrations) <> 45
     OR EXISTS (SELECT 1 FROM jsonb_to_recordset(%s::jsonb) AS e(version bigint,name text,checksum text)
       FULL JOIN public.schema_migrations a USING (version)
       WHERE a.version IS NULL OR e.version IS NULL OR a.name IS DISTINCT FROM e.name
          OR a.checksum IS DISTINCT FROM pg_catalog.decode(e.checksum,'hex')) THEN
    RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE='schema_checksum_mismatch';
  END IF;
`, sharedPostgreSQLLiteral(string(historyJSON)))
	for _, check := range []struct {
		query string
		rows  []map[string]any
	}{
		{sharedPostgreSQLColumnCheck, columns}, {sharedPostgreSQLConstraintCheck, constraints}, {sharedPostgreSQLSequenceCheck, sequences},
	} {
		encoded, err := json.Marshal(check.rows)
		if err != nil {
			return "", migrateError{reason: ReasonDatabaseIncompatible}
		}
		result.WriteString("  SELECT (")
		result.WriteString(strings.Replace(check.query, "$1", sharedPostgreSQLLiteral(string(encoded)), 1))
		result.WriteString(") INTO compatible;\n  IF NOT COALESCE(compatible,FALSE) THEN\n    RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE='database_incompatible';\n  END IF;\n")
	}
	return result.String(), nil
}

// These catalog predicates mirror verifyPostgreSQLHistoryShape and
// verifyPostgreSQLBusinessSchema, consuming their authoritative JSON contract.
// Local drift cases cross-check the generated verifier with native Check.
const sharedPostgreSQLHistoryShapeCheck = `  IF NOT EXISTS (SELECT 1 FROM pg_catalog.pg_class WHERE oid='public.schema_migrations'::regclass AND relkind='r')
     OR (SELECT count(*) FROM information_schema.columns WHERE table_schema='public' AND table_name='schema_migrations') <> 5
     OR EXISTS (SELECT 1 FROM (VALUES
         (1,'version','bigint','NO',0), (2,'name','character varying','NO',255),
         (3,'checksum','bytea','NO',0), (4,'dirty','boolean','NO',0), (5,'applied_at','timestamp with time zone','YES',0)
       ) AS e(position,name,type,nullable,length)
       LEFT JOIN information_schema.columns c ON c.table_schema='public' AND c.table_name='schema_migrations' AND c.ordinal_position=e.position
       WHERE c.column_name IS DISTINCT FROM e.name OR c.data_type IS DISTINCT FROM e.type
          OR c.is_nullable IS DISTINCT FROM e.nullable OR COALESCE(c.character_maximum_length,0) <> e.length)
     OR NOT EXISTS (SELECT 1 FROM pg_catalog.pg_constraint WHERE conrelid='public.schema_migrations'::regclass AND contype='p' AND conkey=ARRAY[1]::smallint[]) THEN
    RAISE EXCEPTION USING ERRCODE='P0001', MESSAGE='database_incompatible';
  END IF;
`

const sharedPostgreSQLColumnCheck = `SELECT COALESCE(bool_and(COALESCE(
 c.relkind='r' AND a.attnum>0 AND NOT a.attisdropped
 AND format_type(a.atttypid,a.atttypmod)=e.data_type AND a.attnotnull=e.not_null
 AND pg_get_expr(d.adbin,d.adrelid) IS NOT DISTINCT FROM e.column_default,FALSE)),FALSE)
 FROM jsonb_to_recordset($1::jsonb) AS e(table_name text,column_name text,data_type text,not_null boolean,column_default text)
 LEFT JOIN pg_namespace n ON n.nspname='public'
 LEFT JOIN pg_class c ON c.relnamespace=n.oid AND c.relname=e.table_name
 LEFT JOIN pg_attribute a ON a.attrelid=c.oid AND a.attname=e.column_name
 LEFT JOIN pg_attrdef d ON d.adrelid=a.attrelid AND d.adnum=a.attnum`

const sharedPostgreSQLConstraintCheck = `SELECT COALESCE(bool_and(COALESCE(
 k.contype::text=e.kind AND pg_get_constraintdef(k.oid,true)=e.definition
 AND k.convalidated=e.validated AND k.condeferrable=e."deferrable"
 AND (k.contype NOT IN ('p','u') OR (i.indisvalid AND i.indisready)),FALSE)),FALSE)
 FROM jsonb_to_recordset($1::jsonb) AS e(table_name text,constraint_name text,kind text,definition text,validated boolean,"deferrable" boolean)
 LEFT JOIN pg_namespace n ON n.nspname='public'
 LEFT JOIN pg_class c ON c.relnamespace=n.oid AND c.relname=e.table_name
 LEFT JOIN pg_constraint k ON k.conrelid=c.oid AND k.conname=e.constraint_name
 LEFT JOIN pg_index i ON i.indexrelid=k.conindid`

const sharedPostgreSQLSequenceCheck = `SELECT COALESCE(bool_and(COALESCE(
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
 LEFT JOIN pg_attribute a ON a.attrelid=t.oid AND a.attnum=d.refobjsubid`
