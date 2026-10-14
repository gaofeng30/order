package adminreport

import "context"

func emptyOrderCounts() map[string]uint64 {
	return map[string]uint64{"RESERVED": 0, "PREPARING": 0, "READY_FOR_PICKUP": 0, "COMPLETED": 0, "REFUNDING": 0, "REFUNDED": 0}
}
func (a *PostgreSQLApplication) OrderCounts(ctx context.Context, user uint64) (map[string]uint64, error) {
	if err := authorizeOwner(ctx, a.db, user); err != nil {
		return nil, err
	}
	rows, err := a.db.QueryContext(ctx, `SELECT state,COUNT(*) FROM orders GROUP BY state`)
	if err != nil {
		return nil, ErrUnavailable
	}
	defer rows.Close()
	counts := emptyOrderCounts()
	for rows.Next() {
		var state string
		var count uint64
		if rows.Scan(&state, &count) != nil {
			return nil, ErrUnavailable
		}
		if _, ok := counts[state]; !ok {
			return nil, ErrUnavailable
		}
		counts[state] = count
	}
	if rows.Err() != nil {
		return nil, ErrUnavailable
	}
	return counts, nil
}
func (a *SharedPGApplication) OrderCounts(ctx context.Context, user uint64) (map[string]uint64, error) {
	reply, err := a.call(ctx, user, "order_admin_report_counts_v1", map[string]any{})
	if err != nil {
		return nil, err
	}
	if reply.Rows == nil || len(reply.Rows) > 6 {
		return nil, ErrUnavailable
	}
	counts := emptyOrderCounts()
	seen := map[string]bool{}
	for _, row := range reply.Rows {
		var state string
		var count uint64
		if sharedReportRow(row).Scan(&state, &count) != nil {
			return nil, ErrUnavailable
		}
		if _, ok := counts[state]; !ok || seen[state] {
			return nil, ErrUnavailable
		}
		seen[state] = true
		counts[state] = count
	}
	return counts, nil
}
