package mysql

import (
	"fmt"

	"github.com/gospacex/gospacex/core/storage/conf"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/plugin/opentelemetry/tracing"
)

var (
	DB *gorm.DB
)

func Init(enableChain bool, mode string, cfg *conf.MysqlConfig) (db *gorm.DB, err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local", cfg.User, cfg.Password, cfg.Ip, cfg.Port, cfg.Db)
	DB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	if enableChain == true { // 设置tracing插件
		if err := DB.Use(tracing.NewPlugin(tracing.WithoutMetrics())); err != nil {
			panic(err)
		}
	}
	if mode == "debug" {
		DB = DB.Debug()
	}
	return DB, err
}
