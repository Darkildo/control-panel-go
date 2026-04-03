package service

import (
	"context"
	"database/sql"
	"io"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "control-panel-go/gen/pb"
	"control-panel-go/internal/models"
	"control-panel-go/internal/repository"
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

func TestConfigService_ApplyConfig(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := zerolog.New(io.Discard)

	deviceRepo := repository.NewDeviceRepository(db)
	configRepo := repository.NewConfigRepository(db)
	svc := NewConfigService(configRepo, deviceRepo, logger)

	device, err := deviceRepo.Create(ctx, &models.Device{
		Hostname: "test-router",
		IP:       "10.0.0.1",
		IsActive: true,
	})
	require.NoError(t, err)

	configResp, err := svc.CreateConfig(ctx, &pb.CreateConfigRequest{
		DeviceId: device.ID,
		Version:  "v1.0.0",
		Content:  "test config content",
	})
	require.NoError(t, err)
	assert.Nil(t, configResp.AppliedAt)

	applyResp, err := svc.ApplyConfig(ctx, &pb.ApplyConfigRequest{
		Id: configResp.Id,
	})
	require.NoError(t, err)
	assert.True(t, applyResp.Success)
	assert.NotNil(t, applyResp.AppliedAt)
}

func TestConfigService_ApplyConfig_NotFound(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := zerolog.New(io.Discard)

	deviceRepo := repository.NewDeviceRepository(db)
	configRepo := repository.NewConfigRepository(db)
	svc := NewConfigService(configRepo, deviceRepo, logger)

	_, err := svc.ApplyConfig(ctx, &pb.ApplyConfigRequest{
		Id: 99999,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestConfigService_CreateConfig_DeviceNotFound(t *testing.T) {
	db := setupTestDB(t)
	ctx := context.Background()
	logger := zerolog.New(io.Discard)

	deviceRepo := repository.NewDeviceRepository(db)
	configRepo := repository.NewConfigRepository(db)
	svc := NewConfigService(configRepo, deviceRepo, logger)

	_, err := svc.CreateConfig(ctx, &pb.CreateConfigRequest{
		DeviceId: 99999,
		Version:  "v1.0.0",
		Content:  "config content",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
