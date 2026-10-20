package store

import (
	"context"
	"fmt"

	"github.com/baddiamond/prive/internal/core"
)

// The Store is the Manager's persistence.
var _ core.Repository = (*Store)(nil)

// ListAccounts implements core.Repository.
func (s *Store) ListAccounts(ctx context.Context) ([]core.Account, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, engine_id, remote_id, display_name, color, position, created_at
		FROM accounts ORDER BY position, created_at`)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	var out []core.Account
	for rows.Next() {
		var (
			a       core.Account
			created int64
		)
		if err := rows.Scan(&a.ID, &a.EngineID, &a.RemoteID, &a.DisplayName, &a.Color, &a.Position, &created); err != nil {
			return nil, fmt.Errorf("list accounts: %w", err)
		}
		a.CreatedAt = fromMillis(created)
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return out, nil
}

// CreateAccount implements core.Repository.
func (s *Store) CreateAccount(ctx context.Context, a core.Account) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO accounts (id, engine_id, remote_id, display_name, color, position, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		a.ID, a.EngineID, a.RemoteID, a.DisplayName, a.Color, a.Position, toMillis(a.CreatedAt))
	if err != nil {
		return fmt.Errorf("create account: %w", err)
	}
	return nil
}

// UpdateAccount implements core.Repository.
func (s *Store) UpdateAccount(ctx context.Context, a core.Account) error {
	res, err := s.db.ExecContext(ctx, `
		UPDATE accounts SET remote_id = ?, display_name = ?, color = ?, position = ?
		WHERE id = ?`,
		a.RemoteID, a.DisplayName, a.Color, a.Position, a.ID)
	if err != nil {
		return fmt.Errorf("update account: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return fmt.Errorf("update account: %w", core.ErrAccountNotFound)
	}
	return nil
}

// DeleteAccount implements core.Repository. Chats and messages go with it (ON DELETE CASCADE).
func (s *Store) DeleteAccount(ctx context.Context, id core.AccountID) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM accounts WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete account: %w", err)
	}
	return nil
}
