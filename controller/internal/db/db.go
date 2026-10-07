package db

import (
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/qualvey/sing-controller/internal/logger"
	"github.com/qualvey/sing-controller/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// DB 是当前数据库连接，保留为导出变量以兼容既有调用方。
// 所有读取请优先使用 Get()：直接解引用会绕开下面这把锁，
// 与 Close / InitDB 的写入形成数据竞争。
var DB *gorm.DB

// dbMu 保护 DB。HTTP handler、后台采集协程与 Engine 关闭流程
// 会并发读写这个全局指针（Engine.Stop 关闭时仍在服务 HTTP 请求）。
var dbMu sync.RWMutex

// ErrDisabled 表示当前 Controller 没有配置可用的数据库连接。
var ErrDisabled = errors.New("database is not configured")

// Enabled 返回数据库是否已初始化。
func Enabled() bool {
	dbMu.RLock()
	defer dbMu.RUnlock()
	return DB != nil
}

// Get 返回当前数据库连接。调用方应处理 ErrDisabled，不要直接解引用全局 DB。
func Get() (*gorm.DB, error) {
	dbMu.RLock()
	defer dbMu.RUnlock()
	if DB == nil {
		return nil, ErrDisabled
	}
	return DB, nil
}

func setDB(database *gorm.DB) {
	dbMu.Lock()
	DB = database
	dbMu.Unlock()
}

// Close 关闭当前数据库连接。
// 先摘掉全局引用再关闭底层连接池：即使 sqlDB.Close 失败，
// 也不能把可能已不可用的连接继续发出去。
func Close() error {
	dbMu.Lock()
	database := DB
	DB = nil
	dbMu.Unlock()
	if database == nil {
		return nil
	}
	sqlDB, err := database.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// InitDB 连接 PostgreSQL 并做兼容迁移。
// 现有数据库表结构直接沿用；仅新增周期字段列（不破坏存量数据）。
// 迁移全部成功后才发布全局 DB；失败时关闭连接池并保持 DB 为 nil，
// 调用方据此判定「数据库不可用」，而不是拿到半初始化连接。
func InitDB(dsn string) error {
	if strings.TrimSpace(dsn) == "" {
		return ErrDisabled
	}
	// 重入场景（如设置热更新）先释放旧连接，避免连接池泄漏。
	_ = Close()

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Warn),
	})
	if err != nil {
		return err
	}
	discard := func(cause error) error {
		if sqlDB, dbErr := database.DB(); dbErr == nil {
			_ = sqlDB.Close()
		}
		return cause
	}

	// 对齐现有 schema：仅建缺失的表（traffic 等），不重建已有表
	if err := database.AutoMigrate(
		&models.User{},
		&models.InboundNode{},
		&models.TrafficLog{},
		&models.TrafficTotal{},
		&models.UserInboundBinding{},
		&models.OutboundNode{},
		&models.NodeProbeStatus{},
		&models.NodeProbeRecord{},
		&models.WebhookDelivery{},
	); err != nil {
		return discard(err)
	}

	setDB(database)
	// 兼容迁移：为存量用户回填周期字段
	if err := backfillCycles(database); err != nil {
		setDB(nil)
		return discard(err)
	}

	sqlDB, err := database.DB()
	if err != nil {
		setDB(nil)
		return discard(err)
	}
	sqlDB.SetMaxOpenConns(20)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	return nil
}

// backfillCycles 存量用户：cycle_start 回退 created_at，cycle_days 默认 30
func backfillCycles(database *gorm.DB) error {
	if err := database.Model(&models.User{}).
		Where("cycle_start IS NULL").
		Update("cycle_start", gorm.Expr("created_at")).Error; err != nil {
		return err
	}
	if err := database.Model(&models.User{}).
		Where("cycle_days IS NULL OR cycle_days = 0").
		Update("cycle_days", models.DefaultCycleDays).Error; err != nil {
		return err
	}
	logger.Info("[DB] cycle fields backfilled")
	return nil
}
