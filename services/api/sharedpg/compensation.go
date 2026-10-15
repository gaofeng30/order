package sharedpg

import (
	_ "embed"
	"regexp"
	"strings"
)

//go:embed compensation.sql
var compensationDefinition string

//go:embed compensation_readiness.sql
var compensationReadinessDefinition string

//go:embed compensation_auth.sql
var compensationAuthenticationDefinition string

// CompensationSQL generates the separately reviewed production scheduler.
// Installation is not implied; the runtime must verify its catalog and platform
// prerequisites before replacing polling workers.
func CompensationSQL(serviceUID string) (string, error) {
	if !regexp.MustCompile(`^[0-9]{1,64}$`).MatchString(serviceUID) {
		return "", ErrDefinitionMismatch
	}
	// Derive the targeted claim from the authoritative notification claim so its
	// validation and lease contract cannot drift into a second implementation.
	subscription, err := source.ReadFile("subscription.sql")
	if err != nil {
		return "", err
	}
	const signature = "CREATE OR REPLACE FUNCTION public.order_subscription_claim_v1(p_cutoff jsonb,p_lease_now jsonb,p_owner_hex text)"
	text := string(subscription)
	if strings.Count(text, signature) != 1 {
		return "", ErrDefinitionMismatch
	}
	claim := text[strings.Index(text, signature):]
	end := strings.Index(claim, "$order_rpc$;")
	if end < 0 {
		return "", ErrDefinitionMismatch
	}
	claim = claim[:end+len("$order_rpc$;")]
	const selection = "SELECT * INTO ob FROM public.notification_outbox WHERE (state='PENDING' AND next_attempt_at<=cutoff_at) OR (state='IN_FLIGHT' AND lease_expires_at<=lease_at) ORDER BY id LIMIT 1 FOR UPDATE SKIP LOCKED;"
	if strings.Count(claim, selection) != 1 {
		return "", ErrDefinitionMismatch
	}
	claim = strings.Replace(claim, signature, "CREATE OR REPLACE FUNCTION public.order_subscription_claim_one_v1(p_outbox numeric,p_cutoff jsonb,p_lease_now jsonb,p_owner_hex text)", 1)
	claim = strings.Replace(claim, selection, "SELECT * INTO ob FROM public.notification_outbox WHERE id=p_outbox AND ((state='PENDING' AND next_attempt_at<=cutoff_at) OR (state='IN_FLIGHT' AND lease_expires_at<=lease_at)) FOR UPDATE SKIP LOCKED;", 1)
	claim += "\nREVOKE ALL ON FUNCTION public.order_subscription_claim_one_v1(numeric,jsonb,jsonb,text) FROM PUBLIC,anon;\nGRANT EXECUTE ON FUNCTION public.order_subscription_claim_one_v1(numeric,jsonb,jsonb,text) TO authenticated;\n"
	definition := strings.ReplaceAll(compensationDefinition+"\n"+compensationAuthenticationDefinition+"\n"+claim+"\n"+compensationReadinessDefinition, "__ORDER_SERVICE_UID__", serviceUID)
	// ExecutePGSql accepts one SQL statement. A DO block keeps all DDL, grants
	// and initial wake registration atomic without relying on session reuse.
	const delimiter = "$order_compensation_definition$"
	if strings.Contains(definition, delimiter) {
		return "", ErrDefinitionMismatch
	}
	return "DO $order_compensation_install$ BEGIN EXECUTE " + delimiter + definition + delimiter + "; END $order_compensation_install$;\n", nil
}
