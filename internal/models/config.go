package models

import (
	"database/sql"
	"time"
)

type Config struct {
	ID        int64        `db:"id"`
	DeviceID  int64        `db:"device_id"`
	Version   string       `db:"version"`
	Content   string       `db:"content"`
	CreatedAt time.Time    `db:"created_at"`
	AppliedAt sql.NullTime `db:"applied_at"`
}
