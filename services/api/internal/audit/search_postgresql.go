package audit

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

type PostgreSQLSearcher struct{ db *sql.DB }

func NewPostgreSQLSearcher(db *sql.DB) *PostgreSQLSearcher { return &PostgreSQLSearcher{db: db} }
func (s *PostgreSQLSearcher) Search(ctx context.Context, userID uint64, f Filter) ([]Entry, uint64, error) {
	var owner uint64
	if s.db.QueryRowContext(ctx, `SELECT id FROM merchant_accounts WHERE bound_user_id=$1 AND enabled=TRUE AND deleted_at IS NULL AND role='OWNER'`, userID).Scan(&owner) != nil {
		return nil, 0, ErrForbidden
	}
	query := `SELECT id,action,COALESCE(target_type,''),COALESCE(target_id::text,''),result,reason_code,COALESCE(encode(request_id_hash,'hex'),''),COALESCE(actor_account_id,0),occurred_at FROM action_audits WHERE id>$1`
	args := []any{f.Page.AfterID}
	if f.Action != "" {
		query += fmt.Sprintf(" AND action=$%d", len(args)+1)
		args = append(args, f.Action)
	}
	if f.Target != "" {
		query += fmt.Sprintf(" AND target_type=$%d", len(args)+1)
		args = append(args, f.Target)
	}
	if f.From != "" {
		query += fmt.Sprintf(" AND occurred_at>=$%d", len(args)+1)
		args = append(args, f.From)
	}
	if f.To != "" {
		query += fmt.Sprintf(" AND occurred_at<$%d", len(args)+1)
		args = append(args, f.To+" 23:59:59.999999")
	}
	query += fmt.Sprintf(" ORDER BY id LIMIT $%d", len(args)+1)
	args = append(args, f.Page.Limit)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, ErrUnavailable
	}
	defer rows.Close()
	out := []Entry{}
	for rows.Next() {
		var e Entry
		var result, reason, requestHash string
		if rows.Scan(&e.ID, &e.Action, &e.TargetType, &e.TargetID, &result, &reason, &requestHash, &e.ActorAccountID, &e.CreatedAt) != nil {
			return nil, 0, ErrUnavailable
		}
		e.ResultCode = result + ":" + reason
		e.RequestID = maskHash(requestHash)
		out = append(out, e)
	}
	if rows.Err() != nil {
		return nil, 0, ErrUnavailable
	}
	next := uint64(0)
	if len(out) == int(f.Page.Limit) {
		next = out[len(out)-1].ID
	}
	return out, next, nil
}
func maskHash(v string) string {
	v = strings.ToLower(v)
	if len(v) < 12 {
		return ""
	}
	return v[:12]
}
