package database

import (
	"fmt"
	"sync"

	"ip-geo/internal/config"
	"ip-geo/internal/logger"

	"github.com/oschwald/maxminddb-golang"
)

// MMDBManager 管理MaxMind数据库连接
type MMDBManager struct {
	ASNDB   *maxminddb.Reader
	CityDB  *maxminddb.Reader
	GeoCNDB *maxminddb.Reader
}

var (
	instance *MMDBManager
	once     sync.Once
)

// GetInstance 获取MMDBManager的单例实例
func GetInstance() *MMDBManager {
	once.Do(func() {
		instance = &MMDBManager{}
	})
	return instance
}

// InitializeDB 初始化数据库连接
func InitializeDB() error {
	logger.Info("开始初始化数据库")

	// 加载配置
	cfg := config.GetInstance()
	if err := cfg.LoadFromFile("config.json"); err != nil {
		logger.Warn("加载配置文件失败，使用默认配置: %v", err)
	}

	// 获取数据库管理器实例
	db := GetInstance()

	// 打开ASN数据库
	logger.Debug("打开ASN数据库: %s", cfg.ASNDB)
	asnDB, err := maxminddb.Open(cfg.ASNDB)
	if err != nil {
		return fmt.Errorf("打开ASN数据库失败: %v", err)
	}
	db.ASNDB = asnDB

	// 打开City数据库
	logger.Debug("打开City数据库: %s", cfg.CityDB)
	cityDB, err := maxminddb.Open(cfg.CityDB)
	if err != nil {
		return fmt.Errorf("打开City数据库失败: %v", err)
	}
	db.CityDB = cityDB

	// 打开GeoCN数据库
	logger.Debug("打开GeoCN数据库: %s", cfg.GeoCNDB)
	geoCNDB, err := maxminddb.Open(cfg.GeoCNDB)
	if err != nil {
		return fmt.Errorf("打开GeoCN数据库失败: %v", err)
	}
	db.GeoCNDB = geoCNDB

	logger.Info("数据库初始化完成")
	return nil
}

// Close 关闭所有数据库连接
func (m *MMDBManager) Close() {
	logger.Info("关闭数据库连接")
	if m.ASNDB != nil {
		if err := m.ASNDB.Close(); err != nil {
			logger.Error("关闭ASN数据库失败: %v", err)
		}
	}
	if m.CityDB != nil {
		if err := m.CityDB.Close(); err != nil {
			logger.Error("关闭City数据库失败: %v", err)
		}
	}
	if m.GeoCNDB != nil {
		if err := m.GeoCNDB.Close(); err != nil {
			logger.Error("关闭GeoCN数据库失败: %v", err)
		}
	}
}
