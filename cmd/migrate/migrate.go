package migrate

import (
	"github.com/jvdiamondtech/ms-notification-cat/cmd"
	"github.com/jvdiamondtech/ms-notification-cat/internal/di"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/database/mysql"
	"github.com/spf13/cobra"
)

// migrateCmd 表示 migrate 命令
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Data migration from legacy system",
	Long:  `Data migration from fatcat_staging database to ms_fatnotificationcat database`,
	Run:   runMigrate,
}

var legacyDSN string

func init() {
	// 添加 legacy DSN 參數
	migrateCmd.Flags().StringVar(&legacyDSN, "legacy-dsn", "",
		"Legacy database DSN (required)\nFormat: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local")
	_ = migrateCmd.MarkFlagRequired("legacy-dsn")

	cmd.AddCommand(migrateCmd)
}

func runMigrate(cobraCmd *cobra.Command, args []string) {
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()

	logger.InfoLog("Starting data migration service...")

	// 初始化主資料庫連接
	database, err := mysql.NewDatabase(cfg, logger)
	if err != nil {
		logger.ErrorLog("Failed to connect to database", logger.Error("error", err))
		return
	}
	defer func() {
		_ = database.Close()
	}()

	db := database.GetDBConnection()

	// 初始化依賴注入
	container, err := di.InitializeMigrateHandler(cfg, logger, db, legacyDSN)
	if err != nil {
		logger.ErrorLog("Failed to initialize migrate handler", logger.Error("error", err))
		return
	}

	// 執行資料搬遷
	if err := container.RunMigration(); err != nil {
		logger.ErrorLog("Data migration failed", logger.Error("error", err))
		return
	}

	logger.InfoLog("Data migration completed successfully")
}
