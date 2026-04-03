package models

import "time"

type Device struct {
	ID        int64     `db:"id"`
	Hostname  string    `db:"hostname"`
	IP        string    `db:"ip"`
	Location  string    `db:"location"`
	IsActive  bool      `db:"is_active"`
	CreatedAt time.Time `db:"created_at"`
}
