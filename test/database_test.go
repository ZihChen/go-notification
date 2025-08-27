// 檔案: tests/database_test.go
package tests

import (
	"testing"

	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/database/mysql"
	"github.com/jvdiamondtech/ms-notification-cat/test/helper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDBConnection(t *testing.T) {
	// 加載配置
	cfg, err := config.LoadConfig()
	require.NoError(t, err, "Should load config without error")

	logger := helper.SetupLoggerMock(t)
	// 初始化資料庫連接
	db, err := mysql.NewDatabase(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to connect to database: %v", err)
	}

	// 測試資料庫連接
	err = db.Ping()
	assert.NoError(t, err, "Should be able to ping the database")
}
