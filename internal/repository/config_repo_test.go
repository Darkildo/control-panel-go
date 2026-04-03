package repository

import (
	"context"
	"database/sql"
	"fmt"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"control-panel-go/internal/models"
)

func setupTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite3", ":memory:?_foreign_keys=on")
	require.NoError(t, err)

	migrations := []string{
		`CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			login TEXT NOT NULL UNIQUE,
			password_hash TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE devices (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			hostname TEXT NOT NULL,
			ip TEXT NOT NULL,
			location TEXT NOT NULL DEFAULT '',
			is_active BOOLEAN NOT NULL DEFAULT 1,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE configs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			device_id INTEGER NOT NULL,
			version TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			applied_at DATETIME,
			FOREIGN KEY (device_id) REFERENCES devices(id) ON DELETE CASCADE
		);`,
	}
	for _, m := range migrations {
		_, err := db.Exec(m)
		require.NoError(t, err)
	}

	t.Cleanup(func() { db.Close() })
	return db
}

func TestConfigRepository_CreateAndGet(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	deviceRepo := NewDeviceRepository(db)
	configRepo := NewConfigRepository(db)

	device, err := deviceRepo.Create(ctx, &models.Device{
		Hostname: "router-01",
		IP:       "192.168.1.1",
		Location: "DC-1",
		IsActive: true,
	})
	require.NoError(t, err)
	require.NotNil(t, device)

	config, err := configRepo.Create(ctx, &models.Config{
		DeviceID: device.ID,
		Version:  "v1.0.0",
		Content:  "interface eth0\n  ip address 10.0.0.1/24",
	})
	require.NoError(t, err)
	require.NotNil(t, config)

	assert.Equal(t, device.ID, config.DeviceID)
	assert.Equal(t, "v1.0.0", config.Version)
	assert.Equal(t, "interface eth0\n  ip address 10.0.0.1/24", config.Content)
	assert.False(t, config.AppliedAt.Valid, "applied_at should be null initially")
	assert.NotZero(t, config.CreatedAt)

	fetched, err := configRepo.GetByID(ctx, config.ID)
	require.NoError(t, err)
	require.NotNil(t, fetched)
	assert.Equal(t, config.ID, fetched.ID)
	assert.Equal(t, config.Version, fetched.Version)
}

func TestConfigRepository_ListWithPagination(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	deviceRepo := NewDeviceRepository(db)
	configRepo := NewConfigRepository(db)

	device, err := deviceRepo.Create(ctx, &models.Device{
		Hostname: "switch-01",
		IP:       "10.0.0.1",
		IsActive: true,
	})
	require.NoError(t, err)

	for i := 1; i <= 5; i++ {
		_, err := configRepo.Create(ctx, &models.Config{
			DeviceID: device.ID,
			Version:  fmt.Sprintf("v%d.0.0", i),
			Content:  fmt.Sprintf("config-%d", i),
		})
		require.NoError(t, err)
	}

	configs, total, err := configRepo.List(ctx, device.ID, 1, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, configs, 2)

	configs, total, err = configRepo.List(ctx, device.ID, 3, 2)
	require.NoError(t, err)
	assert.Equal(t, 5, total)
	assert.Len(t, configs, 1)
}

func TestConfigRepository_Apply(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()

	deviceRepo := NewDeviceRepository(db)
	configRepo := NewConfigRepository(db)

	device, err := deviceRepo.Create(ctx, &models.Device{
		Hostname: "fw-01",
		IP:       "172.16.0.1",
		IsActive: true,
	})
	require.NoError(t, err)

	config, err := configRepo.Create(ctx, &models.Config{
		DeviceID: device.ID,
		Version:  "v1.0.0",
		Content:  "firewall rules...",
	})
	require.NoError(t, err)
	assert.False(t, config.AppliedAt.Valid)

	appliedAt, err := configRepo.Apply(ctx, config.ID)
	require.NoError(t, err)
	assert.False(t, appliedAt.IsZero())

	applied, err := configRepo.GetByID(ctx, config.ID)
	require.NoError(t, err)
	assert.True(t, applied.AppliedAt.Valid)
}
