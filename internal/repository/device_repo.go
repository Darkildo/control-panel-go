package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"control-panel-go/internal/models"
)

type DeviceRepository struct {
	db *sql.DB
}

func NewDeviceRepository(db *sql.DB) *DeviceRepository {
	return &DeviceRepository{db: db}
}

func (r *DeviceRepository) Create(ctx context.Context, device *models.Device) (*models.Device, error) {
	result, err := r.db.ExecContext(ctx,
		`INSERT INTO devices (hostname, ip, location, is_active) VALUES (?, ?, ?, ?)`,
		device.Hostname, device.IP, device.Location, device.IsActive,
	)
	if err != nil {
		return nil, fmt.Errorf("insert device: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("get last insert id: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *DeviceRepository) GetByID(ctx context.Context, id int64) (*models.Device, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, hostname, ip, location, is_active, created_at FROM devices WHERE id = ?`,
		id,
	)

	var d models.Device
	if err := row.Scan(&d.ID, &d.Hostname, &d.IP, &d.Location, &d.IsActive, &d.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("scan device: %w", err)
	}
	return &d, nil
}

func (r *DeviceRepository) List(ctx context.Context, isActive *bool, hostnameSearch string) ([]*models.Device, error) {
	query := `SELECT id, hostname, ip, location, is_active, created_at FROM devices WHERE 1=1`
	var args []interface{}

	if isActive != nil {
		query += ` AND is_active = ?`
		args = append(args, *isActive)
	}

	if hostnameSearch != "" {
		query += ` AND hostname LIKE ?`
		args = append(args, "%"+strings.ReplaceAll(hostnameSearch, "%", "%%")+"%")
	}

	query += ` ORDER BY id DESC`

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query devices: %w", err)
	}
	defer rows.Close()

	var devices []*models.Device
	for rows.Next() {
		var d models.Device
		if err := rows.Scan(&d.ID, &d.Hostname, &d.IP, &d.Location, &d.IsActive, &d.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan device: %w", err)
		}
		devices = append(devices, &d)
	}
	return devices, rows.Err()
}

func (r *DeviceRepository) Update(ctx context.Context, id int64, hostname, ip, location *string, isActive *bool) (*models.Device, error) {
	setClauses := []string{}
	args := []interface{}{}

	if hostname != nil {
		setClauses = append(setClauses, "hostname = ?")
		args = append(args, *hostname)
	}
	if ip != nil {
		setClauses = append(setClauses, "ip = ?")
		args = append(args, *ip)
	}
	if location != nil {
		setClauses = append(setClauses, "location = ?")
		args = append(args, *location)
	}
	if isActive != nil {
		setClauses = append(setClauses, "is_active = ?")
		args = append(args, *isActive)
	}

	if len(setClauses) == 0 {
		return r.GetByID(ctx, id)
	}

	query := "UPDATE devices SET " + strings.Join(setClauses, ", ") + " WHERE id = ?"
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("update device: %w", err)
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

func (r *DeviceRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, `DELETE FROM devices WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if rows == 0 {
		return fmt.Errorf("device not found")
	}
	return nil
}
