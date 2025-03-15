package database

import (
	"context"
	"fmt"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

type MasterDs struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	DBName   string `json:"name,omitempty"`
}

type SlaveDs struct {
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Host     string `json:"host,omitempty"`
	DBName   string `json:"name,omitempty"`
}

type Config struct {
	MasterDataSource      MasterDs      `json:"masterDatasource"`
	SlaveDataSource       SlaveDs       `json:"slaveDatasource"`
	MaxIdleConnections    int           `json:"maxIdleConnections,omitempty"`
	MaxOpenConnections    int           `json:"maxOpenConnections,omitempty"`
	MaxConnectionLifeTime time.Duration `json:"maxConnectionLifeTime,omitempty"`
	MaxConnectionIdleTime time.Duration `json:"maxConnectionIdleTime,omitempty"`
	DisableTLS            bool          `json:"disableTLS,omitempty"`
	Debug                 bool          `json:"debug,omitempty"`
}

type ConnectionMaster struct {
	DB *gorm.DB
}

type ConnectionSlave struct {
	DB *gorm.DB
}

func OpenMaster(ctx context.Context, cfg Config) (*ConnectionMaster, func(), error) {
	logger := ctxzap.Extract(ctx).Sugar()

	defer logger.Infof("master database: connected using user %s at %v", cfg.MasterDataSource.User, cfg.MasterDataSource.Host)

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.MasterDataSource.User, cfg.MasterDataSource.Password, cfg.MasterDataSource.Host, cfg.MasterDataSource.DBName)

	gormLog := gormlogger.Default
	if !cfg.Debug {
		gormLog = gormlogger.Discard
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      gormLog,
	})
	if err != nil {
		return nil, nil, err
	}

	// setting sql DB params
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, errors.Wrap(err, "master database: could not set sql.DB params")
	}
	sqlDB.SetConnMaxIdleTime(cfg.MaxConnectionIdleTime)
	sqlDB.SetConnMaxLifetime(cfg.MaxConnectionLifeTime)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConnections)

	cleanup := func() {
		if err := sqlDB.Close(); err != nil {
			logger.Panicf("master database: failed to close db connections %v", err)
		}
	}

	return &ConnectionMaster{
		DB: db,
	}, cleanup, nil
}

func OpenSlave(ctx context.Context, cfg Config) (*ConnectionSlave, func(), error) {
	logger := ctxzap.Extract(ctx).Sugar()

	defer logger.Infof("slave database: connected using user %s at %v", cfg.SlaveDataSource.User, cfg.SlaveDataSource.Host)

	dsn := fmt.Sprintf("%s:%s@tcp(%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.SlaveDataSource.User, cfg.SlaveDataSource.Password, cfg.SlaveDataSource.Host, cfg.SlaveDataSource.DBName)

	gormLog := gormlogger.Default
	if !cfg.Debug {
		gormLog = gormlogger.Discard
	}
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		PrepareStmt: true,
		Logger:      gormLog,
	})
	if err != nil {
		return nil, nil, err
	}

	// setting sql DB params
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, errors.Wrap(err, "slave database: could not set sql.DB params")
	}
	sqlDB.SetConnMaxIdleTime(cfg.MaxConnectionIdleTime)
	sqlDB.SetConnMaxLifetime(cfg.MaxConnectionLifeTime)
	sqlDB.SetMaxIdleConns(cfg.MaxIdleConnections)
	sqlDB.SetMaxOpenConns(cfg.MaxOpenConnections)

	cleanup := func() {
		if err := sqlDB.Close(); err != nil {
			logger.Panicf("slave database: failed to close db connections %v", err)
		}
	}

	return &ConnectionSlave{
		DB: db,
	}, cleanup, nil
}
