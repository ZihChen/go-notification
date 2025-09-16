package migrate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jvdiamondtech/ms-notification-cat/cmd"
	"github.com/jvdiamondtech/ms-notification-cat/internal/di"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/config"
	"github.com/jvdiamondtech/ms-notification-cat/internal/infrastructure/database/mysql"
	"github.com/spf13/cobra"
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Data migration from legacy system",
	Long:  `Data migration from fatcat_staging database to ms_fatnotificationcat database`,
	Run:   runMigrate,
}

var legacyDSN string

func init() {
	// Add legacy DSN parameter
	migrateCmd.Flags().StringVar(&legacyDSN, "legacy-dsn", "",
		"Legacy database DSN (required)\nFormat: user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local")
	_ = migrateCmd.MarkFlagRequired("legacy-dsn")

	cmd.AddCommand(migrateCmd)
}

func runMigrate(cobraCmd *cobra.Command, args []string) {
	cfg := cmd.GetConfig()
	logger := cmd.GetLogger()

	// Validate Legacy DSN format
	if err := validateLegacyDSN(legacyDSN); err != nil {
		logger.ErrorLog("Invalid legacy DSN format", logger.Error("error", err))
		fmt.Printf("❌ Invalid DSN format: %v\n", err)
		fmt.Println("\n✅ Correct format examples:")
		fmt.Println("  --legacy-dsn \"user:password@tcp(host:port)/dbname?charset=utf8mb4&parseTime=True&loc=Local\"")
		fmt.Println("  --legacy-dsn \"root:pass123@tcp(localhost:3306)/fatcat_staging?charset=utf8mb4&parseTime=True&loc=Local\"")
		return
	}

	logger.InfoLog("Starting data migration service...")
	logger.InfoLog("Legacy DSN validation passed", logger.String("dsn_host", extractHostFromDSN(legacyDSN)))

	// Optional: Add confirmation step (recommended in production environment)
	if !confirmMigration(legacyDSN, cfg) {
		logger.InfoLog("Migration cancelled by user")
		fmt.Println("❌ Migration cancelled")
		return
	}

	// Initialize main database connection
	database, err := mysql.NewDatabase(cfg, logger)
	if err != nil {
		logger.ErrorLog("Failed to connect to database", logger.Error("error", err))
		return
	}
	defer func() {
		_ = database.Close()
	}()

	db := database.GetDBConnection()

	// Initialize dependency injection
	container, err := di.InitializeMigrateHandler(cfg, logger, db, legacyDSN)
	if err != nil {
		logger.ErrorLog("Failed to initialize migrate handler", logger.Error("error", err))
		return
	}

	// Execute data migration
	if err := container.RunMigration(); err != nil {
		logger.ErrorLog("Data migration failed", logger.Error("error", err))
		return
	}

	logger.InfoLog("Data migration completed successfully")
}

// validateLegacyDSN validates Legacy DSN format
func validateLegacyDSN(dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return fmt.Errorf("DSN cannot be empty")
	}

	// Basic format check: user:password@tcp(host:port)/dbname
	dsnPattern := `^[^:]+:[^@]*@tcp\([^:]+:\d+\)/[^?]+(\?.*)?$`
	matched, err := regexp.MatchString(dsnPattern, dsn)
	if err != nil {
		return fmt.Errorf("DSN format validation failed: %w", err)
	}
	if !matched {
		return fmt.Errorf("DSN format does not match MySQL connection string format")
	}

	// Check required query parameters
	parts := strings.Split(dsn, "?")
	if len(parts) > 1 {
		queryParams := parts[1]
		if !strings.Contains(queryParams, "charset=utf8mb4") {
			return fmt.Errorf("DSN missing required parameter: charset=utf8mb4")
		}
		if !strings.Contains(queryParams, "parseTime=True") {
			return fmt.Errorf("DSN missing required parameter: parseTime=True")
		}
	} else {
		return fmt.Errorf("DSN missing query parameters, at least charset=utf8mb4&parseTime=True required")
	}

	return nil
}

// extractHostFromDSN extracts host name from DSN (for logging display)
func extractHostFromDSN(dsn string) string {
	// Simple host name extraction for logging
	atIndex := strings.Index(dsn, "@tcp(")
	if atIndex == -1 {
		return "unknown"
	}

	start := atIndex + 5
	end := strings.Index(dsn[start:], ":")
	if end == -1 {
		return "unknown"
	}

	return dsn[start : start+end]
}

// extractDBNameFromDSN extracts database name from DSN (for logging display)
func extractDBNameFromDSN(dsn string) string {
	// Find )/dbname pattern
	closeParenIndex := strings.Index(dsn, ")/")
	if closeParenIndex == -1 {
		return "unknown"
	}

	// Start from after )/
	start := closeParenIndex + 2
	remaining := dsn[start:]

	// Find ? or end of string
	end := strings.Index(remaining, "?")
	if end == -1 {
		return remaining // No query parameters, entire remaining part is database name
	}

	return remaining[:end]
}

// confirmMigration confirms migration operation (optional feature)
func confirmMigration(legacyDSN string, cfg *config.Config) bool {
	legacyHost := extractHostFromDSN(legacyDSN)
	legacyDB := extractDBNameFromDSN(legacyDSN)

	targetHost := cfg.Database.Host
	targetDB := cfg.Database.DBName

	fmt.Printf("\n⚠️  About to start data migration:\n")
	fmt.Printf("   📂 Legacy source database:\n")
	fmt.Printf("      Host: %s\n", legacyHost)
	fmt.Printf("      Database: %s\n", legacyDB)
	fmt.Printf("   📦 Target database:\n")
	fmt.Printf("      Host: %s\n", targetHost)
	fmt.Printf("      Database: %s\n", targetDB)
	fmt.Printf("   📅 Migration scope: Data from the past three months\n")
	fmt.Printf("   📊 Migration content: notifications + user_notifications\n")
	fmt.Printf("\nDo you want to continue? (y/N): ")

	var response string
	fmt.Scanln(&response)
	response = strings.ToLower(strings.TrimSpace(response))

	return response == "y" || response == "yes"
}
