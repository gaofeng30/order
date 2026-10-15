package migrate

import (
	"bytes"
)

const (
	ReasonDatabaseUnreachable      = "database_unreachable"
	ReasonDatabaseIncompatible     = "database_incompatible"
	ReasonSchemaUninitialized      = "schema_uninitialized"
	ReasonSchemaDirty              = "schema_dirty"
	ReasonSchemaBehind             = "schema_behind"
	ReasonSchemaTooNew             = "schema_too_new"
	ReasonSchemaChecksumMismatch   = "schema_checksum_mismatch"
	ReasonMigrationLockUnavailable = "migration_lock_unavailable"
	ReasonMigrationPreflightFailed = "migration_preflight_failed"
	ReasonStatementFailed          = "migration_statement_failed"
	ReasonMigrationFailed          = "migration_failed"
)

// State is the public non-sensitive schema readiness result.
type State struct {
	Ready          bool
	Reason         string
	CurrentVersion uint64
}

type historyRow struct {
	Version  uint64
	Name     string
	Checksum [32]byte
	Dirty    bool
}

func classifyHistory(exists bool, rows []historyRow, migrations []Migration) State {
	if !exists || len(rows) == 0 {
		return State{Reason: ReasonSchemaUninitialized}
	}
	for index, row := range rows {
		if row.Dirty {
			return State{Reason: ReasonSchemaDirty, CurrentVersion: row.Version}
		}
		if index >= len(migrations) || row.Version != uint64(index+1) {
			return State{Reason: ReasonSchemaTooNew, CurrentVersion: row.Version}
		}
		expected := migrations[index]
		if row.Name != expected.Name || !bytes.Equal(row.Checksum[:], expected.Checksum[:]) {
			return State{Reason: ReasonSchemaChecksumMismatch, CurrentVersion: row.Version}
		}
	}
	current := rows[len(rows)-1].Version
	if len(rows) < len(migrations) {
		return State{Reason: ReasonSchemaBehind, CurrentVersion: current}
	}
	return State{Ready: true, CurrentVersion: current}
}
