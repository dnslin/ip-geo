package config

import (
	"encoding/json"
	"os"
	"sync"

	"ip-geo/internal/logger"
)

// Config 配置结构
type Config struct {
	// 数据库配置
	ASNDB   string `json:"asn_db_path"`
	CityDB  string `json:"city_db_path"`
	GeoCNDB string `json:"geo_cn_db_path"`

	// 服务器配置
	Server struct {
		Port int `json:"port"`
	} `json:"server"`
}

var (
	instance *Config
	once     sync.Once
)

// GetInstance 获取Config的单例实例
func GetInstance() *Config {
	once.Do(func() {
		instance = &Config{
			// 默认配置
			ASNDB:   "mmdb/GeoLite2-ASN.mmdb",
			CityDB:  "mmdb/GeoLite2-City.mmdb",
			GeoCNDB: "mmdb/GeoCN.mmdb",
			Server: struct {
				Port int `json:"port"`
			}{
				Port: 8080,
			},
		}
	})
	return instance
}

// LoadFromFile 从文件加载配置
func (c *Config) LoadFromFile(filename string) error {
	logger.Debug("从文件加载配置: %s", filename)
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(data, c); err != nil {
		return err
	}

	logger.Info("配置加载成功")
	return nil
}
