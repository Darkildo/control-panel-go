package repository

import (
	"context"
	"database/sql"
	"fmt"

	"control-panel-go/internal/models"
)

type UserRepository struct {
	db *sql.DB
}

func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, login, passwordHash string) (*models.User, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO users (login, password_hash) VALUES (?, ?)`,
		login, passwordHash,
	)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *UserRepository) GetByLogin(ctx context.Context, login string) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, login, password_hash, created_at FROM users WHERE login = ?`,
		login,
	)

	var u models.User
	if err := row.Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int64) (*models.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, login, password_hash, created_at FROM users WHERE id = ?`,
		id,
	)

	var u models.User
	if err := row.Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan user: %w", err)
	}
	return &u, nil
}
