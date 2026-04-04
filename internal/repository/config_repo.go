package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"control-panel-go/internal/models"
)

type ConfigRepository struct {
	db *sql.DB
}

func NewConfigRepository(db *sql.DB) *ConfigRepository {
	return &ConfigRepository{db: db}
}

func (r *ConfigRepository) Create(ctx context.Context, config *models.Config) (*models.Config, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO configs (device_id, version, content) VALUES (?, ?, ?)`,
		config.DeviceID, config.Version, config.Content,
	)
	if err != nil {
		return nil, fmt.Errorf("insert config: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *ConfigRepository) GetByID(ctx context.Context, id int64) (*models.Config, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, device_id, version, content, created_at, applied_at FROM configs WHERE id = ?`,
		id,
	)

	var c models.Config
	if err := row.Scan(&c.ID, &c.DeviceID, &c.Version, &c.Content, &c.CreatedAt, &c.AppliedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan config: %w", err)
	}
	return &c, nil
}

func (r *ConfigRepository) List(ctx context.Context, deviceID int64, page, pageSize int) ([]*models.Config, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	var total int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM configs WHERE device_id = ?`,
		deviceID,
	).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count configs: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, device_id, version, content, created_at, applied_at 
		 FROM configs WHERE device_id = ? ORDER BY id DESC LIMIT ? OFFSET ?`,
		deviceID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("query configs: %w", err)
	}
	defer rows.Close()

	var configs []*models.Config
	for rows.Next() {
		var c models.Config
		if err := rows.Scan(&c.ID, &c.DeviceID, &c.Version, &c.Content, &c.CreatedAt, &c.AppliedAt); err != nil {
			return nil, 0, fmt.Errorf("scan config: %w", err)
		}
		configs = append(configs, &c)
	}
	return configs, total, rows.Err()
}

func (r *ConfigRepository) Update(ctx context.Context, id int64, version, content *string) (*models.Config, error) {
	setClauses := []string{}
	args := []interface{}{}

	if version != nil {
		setClauses = append(setClauses, "version = ?")
		args = append(args, *version)
	}
	if content != nil {
		setClauses = append(setClauses, "content = ?")
		args = append(args, *content)
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	query := "UPDATE configs SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update config: %w", err)
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

func (r *ConfigRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM configs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete config: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("config not found")
	}
	return nil
}

func (r *ConfigRepository) Apply(ctx context.Context, id int64) (time.Time, error) {
	now := time.Now().UTC()
	result, err := r.db.ExecContext(ctx,
		`UPDATE configs SET applied_at = ? WHERE id = ?`,
		now, id,
	)
	if err != nil {
		return time.Time{}, fmt.Errorf("apply config: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return time.Time{}, fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return time.Time{}, fmt.Errorf("config not found")
	}
	return now, nil
}
