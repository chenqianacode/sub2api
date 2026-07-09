package repository

import (
	"context"
	"database/sql"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

// GetUsageOverviewSummary gets shared global aggregate usage for the usage overview page.
func (r *usageLogRepository) GetUsageOverviewSummary(ctx context.Context, startTime, endTime, todayStart time.Time) (*usagestats.UsageOverviewSummary, error) {
	query := `
		SELECT
			COUNT(*) AS total_requests,
			(SELECT COUNT(*) FROM users WHERE deleted_at IS NULL) AS total_users,
			COUNT(DISTINCT NULLIF(user_id, 0)) AS active_users,
			(SELECT COUNT(*) FROM accounts WHERE deleted_at IS NULL) AS total_accounts,
			COUNT(DISTINCT NULLIF(account_id, 0)) AS active_accounts,
			COALESCE(SUM(input_tokens), 0) AS input_tokens,
			COALESCE(SUM(output_tokens), 0) AS output_tokens,
			COALESCE(SUM(cache_creation_tokens + cache_read_tokens), 0) AS cache_tokens,
			COALESCE(SUM(input_tokens + output_tokens + cache_creation_tokens + cache_read_tokens), 0) AS total_tokens,
			COALESCE(SUM(total_cost), 0) AS total_cost,
			COALESCE(SUM(actual_cost), 0) AS total_actual_cost,
			COALESCE(SUM(COALESCE(account_stats_cost, total_cost) * COALESCE(account_rate_multiplier, 1)), 0) AS total_account_cost,
			COUNT(*) FILTER (WHERE created_at >= $3) AS today_requests,
			COALESCE(SUM(actual_cost) FILTER (WHERE created_at >= $3), 0) AS today_cost
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2
	`

	stats := &usagestats.UsageOverviewSummary{}
	if err := scanSingleRow(
		ctx,
		r.sql,
		query,
		[]any{startTime, endTime, todayStart},
		&stats.TotalRequests,
		&stats.TotalUsers,
		&stats.ActiveUsers,
		&stats.TotalAccounts,
		&stats.ActiveAccounts,
		&stats.InputTokens,
		&stats.OutputTokens,
		&stats.CacheTokens,
		&stats.TotalTokens,
		&stats.TotalCost,
		&stats.TotalActualCost,
		&stats.TotalAccountCost,
		&stats.TodayRequests,
		&stats.TodayCost,
	); err != nil {
		return nil, err
	}
	return stats, nil
}

func normalizeUsageOverviewPagination(page, pageSize int) (int, int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize, (page - 1) * pageSize
}

// ListDashboardUsageOverviewUsers returns a compact user ranking for the user dashboard.
func (r *usageLogRepository) ListDashboardUsageOverviewUsers(ctx context.Context, todayStart, weekStart, monthStart time.Time, limit int) ([]usagestats.DashboardUsageOverviewUserItem, error) {
	if limit < 1 {
		limit = 6
	}
	if limit > 20 {
		limit = 20
	}

	query := `
		SELECT
			ul.user_id,
			COALESCE(u.username, '') AS username,
			COALESCE(u.email, '') AS email,
			COALESCE(SUM(ul.actual_cost) FILTER (WHERE ul.created_at >= $1), 0) AS today_cost,
			COALESCE(SUM(ul.actual_cost) FILTER (WHERE ul.created_at >= $2), 0) AS week_cost,
			COALESCE(SUM(ul.actual_cost) FILTER (WHERE ul.created_at >= $3), 0) AS month_cost,
			COALESCE(SUM(ul.actual_cost), 0) AS total_cost,
			COUNT(*) FILTER (WHERE ul.created_at >= $1) AS today_requests,
			COUNT(*) FILTER (WHERE ul.created_at >= $2) AS week_requests,
			COUNT(*) FILTER (WHERE ul.created_at >= $3) AS month_requests,
			COUNT(*) AS total_requests,
			MAX(ul.created_at) AS last_used_at
		FROM usage_logs ul
		JOIN users u ON u.id = ul.user_id AND u.deleted_at IS NULL
		WHERE ul.user_id > 0
		GROUP BY ul.user_id, u.username, u.email
		ORDER BY month_cost DESC, total_cost DESC, total_requests DESC, ul.user_id ASC
		LIMIT $4
	`
	rows, err := r.sql.QueryContext(ctx, query, todayStart, weekStart, monthStart, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]usagestats.DashboardUsageOverviewUserItem, 0, limit)
	for rows.Next() {
		var item usagestats.DashboardUsageOverviewUserItem
		var lastUsedAt sql.NullTime
		if err := rows.Scan(
			&item.UserID,
			&item.Username,
			&item.Email,
			&item.TodayCost,
			&item.WeekCost,
			&item.MonthCost,
			&item.TotalCost,
			&item.TodayRequests,
			&item.WeekRequests,
			&item.MonthRequests,
			&item.TotalRequests,
			&lastUsedAt,
		); err != nil {
			return nil, err
		}
		if lastUsedAt.Valid {
			item.LastUsedAt = &lastUsedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// ListUsageOverviewUsers lists aggregate usage grouped by user for the shared overview page.
func (r *usageLogRepository) ListUsageOverviewUsers(ctx context.Context, startTime, endTime, todayStart time.Time, page, pageSize int) ([]usagestats.UsageOverviewUserItem, int64, error) {
	page, pageSize, offset := normalizeUsageOverviewPagination(page, pageSize)

	var total int64
	countQuery := `
		SELECT COUNT(DISTINCT user_id)
		FROM usage_logs
		WHERE created_at >= $1 AND created_at < $2 AND user_id > 0
	`
	if err := scanSingleRow(ctx, r.sql, countQuery, []any{startTime, endTime}, &total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT
			ul.user_id,
			COALESCE(u.username, '') AS username,
			COALESCE(u.email, '') AS email,
			COUNT(*) AS total_requests,
			COALESCE(SUM(ul.input_tokens), 0) AS input_tokens,
			COALESCE(SUM(ul.output_tokens), 0) AS output_tokens,
			COALESCE(SUM(ul.cache_creation_tokens + ul.cache_read_tokens), 0) AS cache_tokens,
			COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) AS total_tokens,
			COALESCE(SUM(ul.total_cost), 0) AS total_cost,
			COALESCE(SUM(ul.actual_cost), 0) AS total_actual_cost,
			COALESCE(SUM(ul.actual_cost) FILTER (WHERE ul.created_at >= $3), 0) AS today_cost,
			MAX(ul.created_at) AS last_used_at
		FROM usage_logs ul
		LEFT JOIN users u ON u.id = ul.user_id
		WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.user_id > 0
		GROUP BY ul.user_id, u.username, u.email
		ORDER BY total_actual_cost DESC, total_requests DESC, ul.user_id ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, todayStart, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]usagestats.UsageOverviewUserItem, 0, pageSize)
	for rows.Next() {
		var item usagestats.UsageOverviewUserItem
		var lastUsedAt sql.NullTime
		if err := rows.Scan(
			&item.UserID,
			&item.Username,
			&item.Email,
			&item.TotalRequests,
			&item.InputTokens,
			&item.OutputTokens,
			&item.CacheTokens,
			&item.TotalTokens,
			&item.TotalCost,
			&item.TotalActualCost,
			&item.TodayCost,
			&lastUsedAt,
		); err != nil {
			return nil, 0, err
		}
		if lastUsedAt.Valid {
			item.LastUsedAt = &lastUsedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// ListUsageOverviewAccounts lists aggregate usage grouped by upstream account for the shared overview page.
func (r *usageLogRepository) ListUsageOverviewAccounts(ctx context.Context, startTime, endTime, todayStart time.Time, page, pageSize int) ([]usagestats.UsageOverviewAccountItem, int64, error) {
	page, pageSize, offset := normalizeUsageOverviewPagination(page, pageSize)

	var total int64
	countQuery := `
		SELECT COUNT(DISTINCT account_id)
		FROM usage_logs ul
		JOIN accounts a ON a.id = ul.account_id AND a.deleted_at IS NULL AND a.status = 'active'
		WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.account_id > 0
	`
	if err := scanSingleRow(ctx, r.sql, countQuery, []any{startTime, endTime}, &total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT
			ul.account_id,
			COALESCE(a.name, '') AS name,
			COALESCE(a.extra->>'email_address', a.extra->>'email', a.credentials->>'email_address', a.credentials->>'email', '') AS email,
			COALESCE(a.platform, '') AS platform,
			COALESCE(a.type, '') AS type,
			COALESCE(a.status, '') AS status,
			COUNT(*) AS total_requests,
			COALESCE(SUM(ul.input_tokens), 0) AS input_tokens,
			COALESCE(SUM(ul.output_tokens), 0) AS output_tokens,
			COALESCE(SUM(ul.cache_creation_tokens + ul.cache_read_tokens), 0) AS cache_tokens,
			COALESCE(SUM(ul.input_tokens + ul.output_tokens + ul.cache_creation_tokens + ul.cache_read_tokens), 0) AS total_tokens,
			COALESCE(SUM(ul.total_cost), 0) AS total_cost,
			COALESCE(SUM(ul.actual_cost), 0) AS total_actual_cost,
			COALESCE(SUM(COALESCE(ul.account_stats_cost, ul.total_cost) * COALESCE(ul.account_rate_multiplier, 1)), 0) AS total_account_cost,
			COALESCE(SUM(COALESCE(ul.account_stats_cost, ul.total_cost) * COALESCE(ul.account_rate_multiplier, 1)) FILTER (WHERE ul.created_at >= $3), 0) AS today_cost,
			MAX(ul.created_at) AS last_used_at
		FROM usage_logs ul
		JOIN accounts a ON a.id = ul.account_id AND a.deleted_at IS NULL AND a.status = 'active'
		WHERE ul.created_at >= $1 AND ul.created_at < $2 AND ul.account_id > 0
		GROUP BY ul.account_id, a.name, a.extra, a.credentials, a.platform, a.type, a.status
		ORDER BY total_account_cost DESC, total_requests DESC, ul.account_id ASC
		LIMIT $4 OFFSET $5
	`
	rows, err := r.sql.QueryContext(ctx, query, startTime, endTime, todayStart, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]usagestats.UsageOverviewAccountItem, 0, pageSize)
	for rows.Next() {
		var item usagestats.UsageOverviewAccountItem
		var lastUsedAt sql.NullTime
		if err := rows.Scan(
			&item.AccountID,
			&item.Name,
			&item.Email,
			&item.Platform,
			&item.Type,
			&item.Status,
			&item.TotalRequests,
			&item.InputTokens,
			&item.OutputTokens,
			&item.CacheTokens,
			&item.TotalTokens,
			&item.TotalCost,
			&item.TotalActualCost,
			&item.TotalAccountCost,
			&item.TodayCost,
			&lastUsedAt,
		); err != nil {
			return nil, 0, err
		}
		if lastUsedAt.Valid {
			item.LastUsedAt = &lastUsedAt.Time
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
