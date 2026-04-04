package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

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

func (r *UserRepository) List(ctx context.Context, loginSearch string) ([]*models.User, error) {
	query := `SELECT id, login, password_hash, created_at FROM users WHERE 1=1`
	var args []interface{}

	if loginSearch != "" {
		query += ` AND login LIKE ?`
		args = append(args, "%"+loginSearch+"%")
	}

	query += ` ORDER BY id DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		var u models.User
		if err := rows.Scan(&u.ID, &u.Login, &u.PasswordHash, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, &u)
	}
	return users, rows.Err()
}

func (r *UserRepository) Update(ctx context.Context, id int64, login *string, passwordHash *string) (*models.User, error) {
	setClauses := []string{}
	args := []interface{}{}

	if login != nil {
		setClauses = append(setClauses, "login = ?")
		args = append(args, *login)
	}
	if passwordHash != nil {
		setClauses = append(setClauses, "password_hash = ?")
		args = append(args, *passwordHash)
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	query := "UPDATE users SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return nil, nil
	}

	return r.GetByID(ctx, id)
}

func (r *UserRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete user: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("user not found")
	}
	return nil
}
