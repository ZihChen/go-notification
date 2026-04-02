package mysql

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/jvdiamondtech/ms-notification-cat/internal/domain/ports/outbound/infrastructure"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/utils/security"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

const (
	dbMaxRetries        = 5
	dbHealthCheckPeriod = 10 * time.Second
	dbFailureThreshold  = 3
	dbPingTimeout       = 5 * time.Second
)

type Database struct {
	dbInstance *gorm.DB
	cfg        *config.Config
	logger     infrastructure.Logger
	isClose    chan struct{}
	mu         sync.RWMutex
}

func NewDatabase(cfg *config.Config, logger infrastructure.Logger) (*Database, error) {
	db := &Database{
		cfg:     cfg,
		logger:  logger,
		isClose: make(chan struct{}),
	}

	// 初始連線帶指數退避重試，讓服務啟動時能等待 DB 就緒
	for attempt := 1; attempt <= dbMaxRetries; attempt++ {
		if err := db.connect(); err != nil {
			if attempt == dbMaxRetries {
				return nil, fmt.Errorf(
					"failed to connect to database after %d attempts: %w",
					dbMaxRetries,
					err,
				)
			}
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			if backoff > 30*time.Second {
				backoff = 30 * time.Second
			}
			logger.WarnLog(
				fmt.Sprintf(
					"Database connection failed (attempt %d/%d), retrying in %v",
					attempt,
					dbMaxRetries,
					backoff,
				),
				logger.Error("err", err),
			)
			time.Sleep(backoff)
			continue
		}
		break
	}

	go db.startHealthChecker()

	return db, nil
}

func (d *Database) connect() error {
	logLevel := logger.Silent
	if d.cfg.App.Debug {
		logLevel = logger.Info
	}

	gormCfg := &gorm.Config{
		PrepareStmt:            false, // 啟用預編譯語句 (語法快取)
		SkipDefaultTransaction: true,  // 關閉默認Transaction，隨應用場景自行加入Transaction
		Logger:                 logger.Default.LogMode(logLevel),
	}

	dsn := buildDSN(d.cfg)
	sanitizedDSN := security.SanitizeDSN(dsn)
	d.logger.InfoLog(fmt.Sprintf("Connecting to database DSN:%s", sanitizedDSN))

	db, err := gorm.Open(mysql.Open(dsn), gormCfg)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection pool: %w", err)
	}

	// 使用配置文件中的連接池設置
	sqlDB.SetMaxIdleConns(d.cfg.Database.MaxIdle)        // 限制最大開啟的連線數
	sqlDB.SetMaxOpenConns(d.cfg.Database.MaxOpen)        // 限制最大閒置連線數
	sqlDB.SetConnMaxLifetime(d.cfg.Database.MaxLifetime) // 連線最大生命週期
	sqlDB.SetConnMaxIdleTime(d.cfg.Database.MaxIdleTime) // 連線最大空閒時間

	d.mu.Lock()
	d.dbInstance = db
	d.mu.Unlock()

	return nil
}

func (d *Database) Close() error {
	// 關閉 Health Checker
	close(d.isClose)

	d.mu.RLock()
	defer d.mu.RUnlock()

	if d.dbInstance != nil {
		sqlDB, err := d.dbInstance.DB()
		if err != nil {
			return fmt.Errorf("DB Connection failed: %w", err)
		}
		return sqlDB.Close()
	}
	return errors.New("DB is empty")
}

func (d *Database) Ping() error {
	d.mu.RLock()
	dbInstance := d.dbInstance
	d.mu.RUnlock()

	sqlDB, err := dbInstance.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), dbPingTimeout)
	defer cancel()

	if err = sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func (d *Database) GetDBConnection() *gorm.DB {
	d.mu.RLock()
	defer d.mu.RUnlock()
	return d.dbInstance
}

// resetConnectionPool 重置連線池，讓 database/sql 在下次 query 時強制重建連線。
// 注意：不替換 *gorm.DB，避免破壞已注入的 repository 引用。
func (d *Database) resetConnectionPool() {
	d.mu.RLock()
	dbInstance := d.dbInstance
	d.mu.RUnlock()

	sqlDB, err := dbInstance.DB()
	if err != nil {
		d.logger.ErrorLog(
			"Failed to get sql.DB for connection pool reset",
			d.logger.Error("err", err),
		)
		return
	}
	// 暫時設置極短的 MaxLifetime，讓連線池清空所有壞連線
	sqlDB.SetConnMaxLifetime(time.Millisecond)
	time.Sleep(100 * time.Millisecond)
	// 恢復原始設置，database/sql 會在下次查詢時自動重建連線
	sqlDB.SetConnMaxLifetime(d.cfg.Database.MaxLifetime)
	d.logger.InfoLog("Database connection pool reset, will reconnect on next query")
}

func buildDSN(cfg *config.Config) string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?parseTime=True&tls=true",
		cfg.Database.User,
		cfg.Database.Password,
		cfg.Database.Host,
		cfg.Database.Port,
		cfg.Database.DBName,
	)
}

func (d *Database) startHealthChecker() {
	ticker := time.NewTicker(dbHealthCheckPeriod)
	defer ticker.Stop()

	consecutiveFailures := 0

	for {
		select {
		case <-ticker.C:
			if err := d.Ping(); err != nil {
				consecutiveFailures++
				d.logger.ErrorLog(
					"Database health check failed",
					d.logger.Error("err", err),
					d.logger.Int("consecutive_failures", consecutiveFailures),
				)

				// 連續失敗達到閾值，重置連線池觸發 database/sql 重連
				if consecutiveFailures >= dbFailureThreshold {
					d.logger.WarnLog(
						"Database connection degraded, resetting connection pool to trigger reconnect",
						d.logger.Int("consecutive_failures", consecutiveFailures),
					)
					d.resetConnectionPool()
				}
			} else {
				if consecutiveFailures > 0 {
					d.logger.InfoLog(
						"Database connection recovered",
						d.logger.Int("previous_failures", consecutiveFailures),
					)
					consecutiveFailures = 0
				}

				// 連線正常，輸出連接池狀態監控
				d.mu.RLock()
				dbInstance := d.dbInstance
				d.mu.RUnlock()

				sqlDB, err := dbInstance.DB()
				if err == nil {
					stats := sqlDB.Stats()

					// 輸出詳細的連接池統計信息
					d.logger.InfoLog("Database connection pool stats",
						d.logger.Int("max_open_connections", stats.MaxOpenConnections),
						d.logger.Int("open_connections", stats.OpenConnections),
						d.logger.Int("in_use", stats.InUse),
						d.logger.Int("idle", stats.Idle),
						d.logger.Int64("wait_count", stats.WaitCount),
						d.logger.String("wait_duration", stats.WaitDuration.String()),
						d.logger.Int64("max_idle_closed", stats.MaxIdleClosed),
						d.logger.Int64("max_lifetime_closed", stats.MaxLifetimeClosed),
					)

					// 輸出GORM配置狀態（用於驗證PrepareStmt設置）
					d.logger.InfoLog(
						"Database configuration status",
						d.logger.Bool("prepare_stmt_enabled", dbInstance.PrepareStmt),
						d.logger.Bool(
							"skip_default_transaction",
							dbInstance.SkipDefaultTransaction,
						),
						d.logger.Int("max_idle_conns_config", d.cfg.Database.MaxIdle),
						d.logger.Int("max_open_conns_config", d.cfg.Database.MaxOpen),
						d.logger.String("max_lifetime_config", d.cfg.Database.MaxLifetime.String()),
						d.logger.String(
							"max_idle_time_config",
							d.cfg.Database.MaxIdleTime.String(),
						),
					)

					// 檢測異常情況並發出警告
					if stats.WaitCount > 0 {
						d.logger.WarnLog(
							"Database connection pool is experiencing waits",
							d.logger.Int64("wait_count", stats.WaitCount),
							d.logger.String("total_wait_duration", stats.WaitDuration.String()),
							d.logger.String(
								"suggestion",
								"Consider increasing DB_MAX_OPEN connections",
							),
						)
					}

					// 檢查是否有過多的空閒連接被關閉
					if stats.MaxIdleClosed > 100 {
						d.logger.WarnLog(
							"High number of idle connections being closed",
							d.logger.Int64("max_idle_closed", stats.MaxIdleClosed),
							d.logger.String(
								"suggestion",
								"Consider increasing DB_MAX_IDLE or reducing DB_MAX_IDLE_TIME",
							),
						)
					}

					// 計算連接池使用率
					if stats.MaxOpenConnections > 0 {
						utilizationPercent := float64(
							stats.OpenConnections,
						) / float64(
							stats.MaxOpenConnections,
						) * 100
						d.logger.InfoLog("Database connection pool utilization",
							d.logger.Float64("utilization_percent", utilizationPercent),
						)

						// 如果使用率超過80%，發出警告
						if utilizationPercent > 80 {
							d.logger.WarnLog(
								"Database connection pool utilization is high",
								d.logger.Float64("utilization_percent", utilizationPercent),
								d.logger.String(
									"suggestion",
									"Connection pool may be under pressure",
								),
							)
						}
					}
				}

				d.logger.InfoLog("Database connection is up!")
			}
		case <-d.isClose:
			return
		}
	}
}
