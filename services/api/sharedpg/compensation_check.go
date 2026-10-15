package sharedpg

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"regexp"
	"slices"
	"strings"
)

type compensationAccess struct {
	Name, Owner         string
	PublicAccess        bool `json:"public_access"`
	AnonAccess          bool `json:"anon_access"`
	AuthenticatedAccess bool `json:"authenticated_access"`
}
type compensationColumn struct {
	Name, Type string
	NotNull    bool `json:"not_null"`
	Default    *string
}
type compensationRelation struct {
	compensationAccess
	Kind, Persistence    string
	RLS                  bool
	ForceRLS             bool `json:"force_rls"`
	Columns              []compensationColumn
	Constraints, Indexes []string
}
type compensationCatalog struct {
	Role, UID       string
	Functions       []function
	Schemas         []compensationAccess
	PlatformSchemas []compensationAccess `json:"platform_schemas"`
	Relations       []compensationRelation
	Triggers        []struct {
		Name, Table, Enabled, Function, Arguments string
		Type                                      int
		When                                      *string
	}
	Extensions []struct{ Name, Version, Schema, Owner string }
}

// CheckCompensation is an additional readiness gate, not a replacement for the
// relational schema, business RPC and environment checks in Check.
func CheckCompensation(ctx context.Context, client Executor, uid string) error {
	definition, err := CompensationSQL(uid)
	if ctx == nil || client == nil || err != nil {
		return ErrDefinitionMismatch
	}
	var raw json.RawMessage
	if client.CallRPC(ctx, "order_compensation_readiness_v1", map[string]any{}, &raw) != nil || verifyCompensationObjects(raw, definition, uid) != nil {
		return ErrDefinitionMismatch
	}
	var c compensationCatalog
	if json.Unmarshal(raw, &c) != nil || len(c.Extensions) != 3 {
		return ErrDefinitionMismatch
	}
	for i, e := range []struct{ name, version, schema string }{{"pg_cron", "1.6", "pg_catalog"}, {"pgcrypto", "1.3", "public"}, {"tencentdb_scf", "1.1", "tencentdb_scf"}} {
		a := c.Extensions[i]
		if a.Name != e.name || a.Version != e.version || a.Schema != e.schema || a.Owner != c.Functions[0].Owner {
			return ErrDefinitionMismatch
		}
	}
	return nil
}

// Object validation is separate from extension availability so isolated native
// PG tests can exercise the catalog using explicit transport stubs. Production
// readiness must additionally require the actual platform extension catalog.
func verifyCompensationObjects(raw []byte, definition, uid string) error {
	var c compensationCatalog
	if json.Unmarshal(raw, &c) != nil || c.Role != "authenticated" || c.UID != uid || len(c.Functions) != 8 || len(c.Schemas) != 1 || len(c.Relations) != 2 || len(c.Triggers) != 3 {
		return ErrDefinitionMismatch
	}
	matches := regexp.MustCompile(`(?s)CREATE (?:OR REPLACE )?FUNCTION ((?:public|order_compensation)\.[a-z_0-9]+)\(.*?AS \$(?:wake|order_rpc)\$(.*?)\$(?:wake|order_rpc)\$;`).FindAllStringSubmatch(definition, -1)
	if len(matches) != 8 {
		return ErrDefinitionMismatch
	}
	bodies := make(map[string]string, len(matches))
	for _, m := range matches {
		if _, ok := bodies[m[1]]; ok {
			return ErrDefinitionMismatch
		}
		bodies[m[1]] = m[2]
	}
	expected := []struct {
		name, types, result string
		args                []string
		definer             bool
	}{
		{"order_compensation.changed", "", "trigger", nil, true},
		{"order_compensation.dispatch", "25 1700", "void", []string{"p_kind", "p_id"}, true},
		{"order_compensation.refresh", "25 1700", "void", []string{"p_kind", "p_id"}, true},
		{"public.order_compensation_authenticate_v1", "", "jsonb", nil, true},
		{"public.order_compensation_claim_v1", "25", "jsonb", []string{"p_token"}, true},
		{"public.order_compensation_finish_v1", "25 16", "jsonb", []string{"p_token", "p_success"}, true},
		{"public.order_compensation_readiness_v1", "", "jsonb", nil, false},
		{"public.order_subscription_claim_one_v1", "1700 3802 3802 25", "jsonb", []string{"p_outbox", "p_cutoff", "p_lease_now", "p_owner_hex"}, false},
	}
	owner := c.Functions[0].Owner
	if owner == "" || owner == "anon" || owner == "authenticated" {
		return ErrDefinitionMismatch
	}
	for i, f := range c.Functions {
		e := expected[i]
		body, ok := bodies[f.Name]
		hash := sha256.Sum256([]byte(body))
		config := []string{"search_path=pg_catalog"}
		if !e.definer {
			config = []string{"search_path=pg_catalog, public", "TimeZone=UTC"}
		}
		if !ok || f.Name != e.name || f.Hash != hex.EncodeToString(hash[:]) || f.Owner != owner || f.SecurityDefiner != e.definer || f.ReturnsSet || f.Strict || f.Volatility != "v" || f.Parallel != "u" || f.ResultType != e.result || f.ArgumentTypes != e.types || !slices.Equal(f.ArgumentNames, e.args) || f.Language != "plpgsql" || f.Kind != "f" || f.Defaults != 0 || f.Variadic != 0 || !slices.Equal(f.Config, config) || f.PublicExecute || f.AnonExecute || f.AuthenticatedExecute != strings.HasPrefix(e.name, "public.") {
			return ErrDefinitionMismatch
		}
	}
	private := func(a compensationAccess) bool {
		return a.Owner == owner && !a.PublicAccess && !a.AnonAccess && !a.AuthenticatedAccess
	}
	if c.Schemas[0].Name != "order_compensation" || !private(c.Schemas[0]) {
		return ErrDefinitionMismatch
	}
	if len(c.PlatformSchemas) != 2 {
		return ErrDefinitionMismatch
	}
	for i, name := range []string{"cron", "tencentdb_scf"} {
		if c.PlatformSchemas[i].Name != name || !private(c.PlatformSchemas[i]) {
			return ErrDefinitionMismatch
		}
	}
	a := c.Relations[0]
	if a.Name != "authentication" || a.Kind != "r" || a.Persistence != "p" || a.RLS || a.ForceRLS || !private(a.compensationAccess) || len(a.Columns) != 2 ||
		a.Columns[0].Name != "singleton" || a.Columns[0].Type != "boolean" || !a.Columns[0].NotNull || a.Columns[0].Default != nil ||
		a.Columns[1].Name != "expires_at" || a.Columns[1].Type != "timestamp with time zone" || !a.Columns[1].NotNull || a.Columns[1].Default != nil ||
		!slices.Equal(a.Constraints, []string{"PRIMARY KEY (singleton)", "CHECK (singleton)"}) ||
		!slices.Equal(a.Indexes, []string{"CREATE UNIQUE INDEX authentication_pkey ON order_compensation.authentication USING btree (singleton)"}) {
		return ErrDefinitionMismatch
	}
	r := c.Relations[1]
	if r.Name != "wakes" || r.Kind != "r" || r.Persistence != "p" || r.RLS || r.ForceRLS || !private(r.compensationAccess) || len(r.Columns) != 8 {
		return ErrDefinitionMismatch
	}
	for i, e := range []struct {
		name, typ string
		notNull   bool
	}{
		{"kind", "text", true}, {"entity_id", "numeric(20,0)", true}, {"next_at", "timestamp with time zone", false}, {"token_hash", "bytea", false}, {"token_until", "timestamp with time zone", false}, {"request_id", "bigint", false}, {"claimed", "boolean", true}, {"retry_after", "timestamp with time zone", false},
	} {
		a := r.Columns[i]
		if a.Name != e.name || a.Type != e.typ || a.NotNull != e.notNull {
			return ErrDefinitionMismatch
		}
		if e.name == "claimed" {
			if a.Default == nil || *a.Default != "false" {
				return ErrDefinitionMismatch
			}
		} else if a.Default != nil {
			return ErrDefinitionMismatch
		}
	}
	if !slices.Equal(r.Constraints, []string{
		"CHECK (((entity_id >= (1)::numeric) AND (entity_id <= '18446744073709551615'::numeric)))",
		"CHECK ((kind = ANY (ARRAY['payment'::text, 'refund'::text, 'subscription'::text])))",
		"PRIMARY KEY (kind, entity_id)",
		"CHECK (((token_hash IS NULL) OR (octet_length(token_hash) = 32)))",
	}) || !slices.Equal(r.Indexes, []string{
		"CREATE UNIQUE INDEX order_compensation_token_unique ON order_compensation.wakes USING btree (token_hash) WHERE (token_hash IS NOT NULL)",
		"CREATE UNIQUE INDEX wakes_pkey ON order_compensation.wakes USING btree (kind, entity_id)",
	}) {
		return ErrDefinitionMismatch
	}
	for i, kind := range []string{"payment", "refund", "subscription"} {
		t := c.Triggers[i]
		if t.Name != "order_compensation_"+kind || t.Table != []string{"prepayments", "refunds", "notification_outbox"}[i] || t.Enabled != "O" || t.Type != 29 || t.Function != "order_compensation.changed" || t.Arguments != hex.EncodeToString([]byte(kind+"\x00")) || t.When != nil {
			return ErrDefinitionMismatch
		}
	}
	return nil
}
