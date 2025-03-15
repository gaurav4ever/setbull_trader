package database

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

type MigrationHandler struct {
	master *ConnectionMaster
	config Config
	logger *zap.Logger
}

func NewMigrationHandler(master *ConnectionMaster, config Config) *MigrationHandler {
	return &MigrationHandler{
		master: master,
		config: config,
		logger: zap.NewNop(),
	}
}

func (m *MigrationHandler) ApplyMigrations() error {
	m.logger.Info("╔═════════════════════════════════════════╗")
	m.logger.Info("║      STARTING DATABASE MIGRATION        ║")
	m.logger.Info("╚═════════════════════════════════════════╝")

	// Get the DSN string
	dsnConfig := fmt.Sprintf("mysql://%s", m.buildDSN())
	maskedDSN := m.getMaskedDSN(dsnConfig)

	m.logger.Info("Step 1: Initializing database connection",
		zap.String("database", m.config.MasterDataSource.DBName),
		zap.String("host", m.config.MasterDataSource.Host),
		zap.String("dsn", maskedDSN),
	)

	// Get the migrations path
	migrationsPath, err := m.getMigrationPath()
	if err != nil {
		m.logger.Error("Failed to get migrations path", zap.Error(err))
		return errors.Wrap(err, "failed to get migrations path")
	}

	m.logger.Info("Step 2: Creating migration instance",
		zap.String("migrations_path", migrationsPath),
	)

	// Setup the migration instance
	migration, err := migrate.New(
		migrationsPath,
		dsnConfig,
	)
	if err != nil {
		m.logger.Error("Failed to create migration instance", zap.Error(err))
		return errors.Wrap(err, "failed to create migration instance")
	}
	defer migration.Close()

	// Get current version before migration
	version, dirty, err := migration.Version()
	if err != nil && err != migrate.ErrNilVersion {
		m.logger.Warn("Could not get current migration version",
			zap.Error(err),
		)
	} else {
		m.logger.Info("Current database state",
			zap.Uint("version", version),
			zap.Bool("dirty", dirty),
		)
	}

	m.logger.Info("Step 3: Applying pending migrations...")

	// Apply all up migrations
	if err := migration.Up(); err != nil {
		if err == migrate.ErrNoChange {
			m.logger.Info("► Database is up to date, no migrations needed")
		} else {
			m.logger.Error("Migration failed", zap.Error(err))
			return errors.Wrap(err, "failed to apply migrations")
		}
	} else {
		// Get new version after migration
		newVersion, newDirty, verErr := migration.Version()
		if verErr == nil {
			m.logger.Info("► Successfully applied migrations",
				zap.Uint("from_version", version),
				zap.Uint("to_version", newVersion),
				zap.Bool("dirty", newDirty),
			)
		}
	}

	m.logger.Info("╔═════════════════════════════════════════╗")
	m.logger.Info("║      MIGRATION PROCESS COMPLETED        ║")
	m.logger.Info("╚═════════════════════════════════════════╝")

	return nil
}

func (m *MigrationHandler) buildDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s)/%s?multiStatements=true&parseTime=true",
		m.config.MasterDataSource.User,
		m.config.MasterDataSource.Password,
		m.config.MasterDataSource.Host,
		m.config.MasterDataSource.DBName,
	)
}

func (m *MigrationHandler) getMigrationPath() (string, error) {
	// Get the working directory
	wd, err := os.Getwd()
	if err != nil {
		return "", errors.Wrap(err, "failed to get working directory")
	}

	// Construct the migrations path
	migrationsPath := filepath.Join(wd, "pkg/database/migrations")

	// Log the migrations path for debugging
	m.logger.Info("Migrations path constructed",
		zap.String("migrations_path", migrationsPath),
	)

	// Verify the migrations directory exists
	if _, err := os.Stat(migrationsPath); os.IsNotExist(err) {
		return "", errors.Wrap(err, "migrations directory not found")
	}

	// List all migration files
	files, err := os.ReadDir(migrationsPath)
	if err != nil {
		return "", errors.Wrap(err, "failed to read migrations directory")
	}

	// Log found migration files
	m.logger.Info("found migration files",
		zap.Int("count", len(files)),
		zap.String("path", migrationsPath),
	)

	for _, file := range files {
		m.logger.Debug("migration file found",
			zap.String("filename", file.Name()),
		)
	}

	// Return the file URL format path
	return fmt.Sprintf("file://%s", migrationsPath), nil
}

// RollbackMigration rolls back the last applied migration
func (m *MigrationHandler) RollbackMigration() error {
	m.logger.Info("╔═════════════════════════════════════════╗")
	m.logger.Info("║     STARTING SINGLE STEP ROLLBACK      ║")
	m.logger.Info("╚═════════════════════════════════════════╝")

	// Get the DSN string with mysql prefix
	dsnConfig := fmt.Sprintf("mysql://%s", m.buildDSN())
	maskedDSN := m.getMaskedDSN(dsnConfig)

	m.logger.Info("Step 1: Initializing database connection",
		zap.String("database", m.config.MasterDataSource.DBName),
		zap.String("host", m.config.MasterDataSource.Host),
		zap.String("dsn", maskedDSN),
	)

	// Get the migrations path
	migrationsPath, err := m.getMigrationPath()
	if err != nil {
		m.logger.Error("Failed to get migrations path", zap.Error(err))
		return errors.Wrap(err, "failed to get migrations path")
	}

	m.logger.Info("Step 2: Creating migration instance",
		zap.String("migrations_path", migrationsPath),
	)

	// Setup the migration instance
	migration, err := migrate.New(
		migrationsPath,
		dsnConfig,
	)
	if err != nil {
		m.logger.Error("Failed to create migration instance", zap.Error(err))
		return errors.Wrap(err, "failed to create migration instance")
	}
	defer func() {
		srcErr, dbErr := migration.Close()
		if srcErr != nil {
			m.logger.Error("Failed to close migration source", zap.Error(srcErr))
		}
		if dbErr != nil {
			m.logger.Error("Failed to close migration database", zap.Error(dbErr))
		}
	}()

	// Get current version before rollback
	version, dirty, err := migration.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			m.logger.Info("► Database is already at base version (no migrations applied)")
			return nil
		}
		m.logger.Error("Failed to get current migration version", zap.Error(err))
		return errors.Wrap(err, "failed to get current migration version")
	}

	m.logger.Info("Step 3: Current database state",
		zap.Uint("version", version),
		zap.Bool("dirty", dirty),
	)

	if version == 0 {
		m.logger.Info("► Database is at base version, no rollback needed")
		return nil
	}

	m.logger.Info("Step 4: Beginning rollback of last migration",
		zap.Uint("current_version", version),
		zap.String("operation", "single step rollback"),
	)

	// Roll back one step
	if err := migration.Steps(-1); err != nil {
		if err == migrate.ErrNoChange {
			m.logger.Info("► No changes to rollback")
		} else {
			m.logger.Error("Rollback failed",
				zap.Error(err),
				zap.Uint("from_version", version),
			)
			return errors.Wrap(err, "failed to rollback migration")
		}
	}

	// Get new version after rollback
	newVersion, dirty, err := migration.Version()
	if err != nil {
		m.logger.Warn("Could not verify final version",
			zap.Error(err),
			zap.Uint("last_known_version", version),
		)
	} else {
		m.logger.Info("► Rollback completed successfully",
			zap.Uint("from_version", version),
			zap.Uint("to_version", newVersion),
			zap.Bool("dirty", dirty),
		)

		// Log affected migration file
		m.logger.Info("Migration rolled back",
			zap.String("file", fmt.Sprintf("%d_*.down.sql", version)),
		)
	}

	m.logger.Info("╔═════════════════════════════════════════╗")
	m.logger.Info("║     SINGLE STEP ROLLBACK COMPLETE      ║")
	m.logger.Info("╚═════════════════════════════════════════╝")

	return nil
}

// RollbackAll rolls back all migrations
func (m *MigrationHandler) RollbackAll() error {
	m.logger.Info("╔═════════════════════════════════════════╗")
	m.logger.Info("║    STARTING COMPLETE MIGRATION RESET    ║")
	m.logger.Info("╚═════════════════════════════════════════╝")

	dsnConfig := fmt.Sprintf("mysql://%s", m.buildDSN())
	maskedDSN := m.getMaskedDSN(dsnConfig)

	m.logger.Info("Step 1: Initializing database connection",
		zap.String("database", m.config.MasterDataSource.DBName),
		zap.String("host", m.config.MasterDataSource.Host),
		zap.String("dsn", maskedDSN),
	)

	migrationsPath, err := m.getMigrationPath()
	if err != nil {
		m.logger.Error("Failed to get migrations path", zap.Error(err))
		return errors.Wrap(err, "failed to get migrations path")
	}

	m.logger.Info("Step 2: Creating migration instance",
		zap.String("migrations_path", migrationsPath),
	)

	migration, err := migrate.New(
		migrationsPath,
		dsnConfig,
	)
	if err != nil {
		m.logger.Error("Failed to create migration instance", zap.Error(err))
		return errors.Wrap(err, "failed to create migration instance")
	}
	defer func() {
		srcErr, dbErr := migration.Close()
		if srcErr != nil {
			m.logger.Error("Failed to close migration source", zap.Error(srcErr))
		}
		if dbErr != nil {
			m.logger.Error("Failed to close migration database", zap.Error(dbErr))
		}
	}()

	// Get current version before rollback
	version, dirty, err := migration.Version()
	if err != nil {
		if err == migrate.ErrNilVersion {
			m.logger.Info("► Database is already at base version (no migrations applied)")
			return nil
		}
		m.logger.Error("Failed to get current migration version", zap.Error(err))
		return errors.Wrap(err, "failed to get current migration version")
	}

	m.logger.Info("Step 3: Current database state",
		zap.Uint("version", version),
		zap.Bool("dirty", dirty),
	)

	if version == 0 {
		m.logger.Info("► Database is already at base version")
		return nil
	}

	m.logger.Info("Step 4: Beginning complete rollback",
		zap.Uint("from_version", version),
		zap.String("target", "BASE VERSION (0)"),
	)

	// Roll back all migrations
	if err := migration.Down(); err != nil {
		if err == migrate.ErrNoChange {
			m.logger.Info("► No changes to rollback")
		} else {
			m.logger.Error("Complete rollback failed", zap.Error(err))
			return errors.Wrap(err, "failed to rollback all migrations")
		}
	}

	// Verify final state
	finalVersion, dirty, err := migration.Version()
	if err == nil {
		m.logger.Info("► Rollback completed successfully",
			zap.Uint("initial_version", version),
			zap.Uint("final_version", finalVersion),
			zap.Bool("dirty", dirty),
		)
	}

	m.logger.Info("╔═════════════════════════════════════════╗")
	m.logger.Info("║    COMPLETE ROLLBACK SUCCESSFUL         ║")
	m.logger.Info("╚═════════════════════════════════════════╝")

	return nil
}

// RollbackTo rolls back migrations to a specific version
func (m *MigrationHandler) RollbackTo(targetVersion uint) error {
	m.logger.Info("╔═════════════════════════════════════════╗")
	m.logger.Info("║    STARTING MIGRATION ROLLBACK TO       ║")
	m.logger.Info("╚═════════════════════════════════════════╝")

	dsnConfig := fmt.Sprintf("mysql://%s", m.buildDSN())
	maskedDSN := m.getMaskedDSN(dsnConfig)

	m.logger.Info("Step 1: Initializing database connection",
		zap.String("database", m.config.MasterDataSource.DBName),
		zap.String("host", m.config.MasterDataSource.Host),
		zap.String("dsn", maskedDSN),
	)

	migrationsPath, err := m.getMigrationPath()
	if err != nil {
		m.logger.Error("Failed to get migrations path", zap.Error(err))
		return errors.Wrap(err, "failed to get migrations path")
	}

	m.logger.Info("Step 2: Creating migration instance",
		zap.String("migrations_path", migrationsPath),
		zap.Uint("target_version", targetVersion),
	)

	migration, err := migrate.New(
		migrationsPath,
		dsnConfig,
	)
	if err != nil {
		m.logger.Error("Failed to create migration instance", zap.Error(err))
		return errors.Wrap(err, "failed to create migration instance")
	}
	defer func() {
		srcErr, dbErr := migration.Close()
		if srcErr != nil {
			m.logger.Error("Failed to close migration source", zap.Error(srcErr))
		}
		if dbErr != nil {
			m.logger.Error("Failed to close migration database", zap.Error(dbErr))
		}
	}()

	// Get current version
	currentVersion, dirty, err := migration.Version()
	if err != nil && err != migrate.ErrNilVersion {
		m.logger.Error("Failed to get current migration version", zap.Error(err))
		return errors.Wrap(err, "failed to get current migration version")
	}

	m.logger.Info("Step 3: Current database state",
		zap.Uint("current_version", currentVersion),
		zap.Bool("dirty", dirty),
		zap.Uint("target_version", targetVersion),
	)

	if currentVersion == targetVersion {
		m.logger.Info("► Database is already at target version",
			zap.Uint("version", targetVersion),
		)
		return nil
	}

	m.logger.Info("Step 4: Beginning rollback process",
		zap.Uint("from_version", currentVersion),
		zap.Uint("to_version", targetVersion),
	)

	// Migrate to specific version
	if err := migration.Migrate(targetVersion); err != nil {
		if err == migrate.ErrNoChange {
			m.logger.Info("► No migration needed, database is at desired version")
		} else {
			m.logger.Error("Rollback failed",
				zap.Error(err),
				zap.Uint("attempted_version", targetVersion),
			)
			return errors.Wrap(err, "failed to migrate to target version")
		}
	}

	// Verify final version
	finalVersion, dirty, err := migration.Version()
	if err == nil {
		m.logger.Info("► Rollback completed successfully",
			zap.Uint("initial_version", currentVersion),
			zap.Uint("final_version", finalVersion),
			zap.Bool("dirty", dirty),
		)
	}

	m.logger.Info("╔═════════════════════════════════════════╗")
	m.logger.Info("║      ROLLBACK PROCESS COMPLETED         ║")
	m.logger.Info("╚═════════════════════════════════════════╝")

	return nil
}

// getMaskedDSN returns a DSN string with sensitive information masked
func (m *MigrationHandler) getMaskedDSN(dsn string) string {
	maskedDSN := dsn
	if m.config.MasterDataSource.Password != "" {
		maskedDSN = strings.Replace(
			maskedDSN,
			m.config.MasterDataSource.Password,
			"*****",
			1,
		)
	}
	return maskedDSN
}
