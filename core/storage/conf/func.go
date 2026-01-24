package conf

import (
	"fmt"

	"github.com/spf13/viper"
)

func ParseConfig(filePath string) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(filePath)
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil { // Handle errors reading the config file
		panic(fmt.Errorf("fatal error config file: %s", err))
	}

	if err = viper.Unmarshal(&Cfg); err != nil {
		panic(fmt.Sprintf("parse config from config.yaml failed:%s", err))
	}
	if err = viper.Unmarshal(&Cfg); err != nil {
		panic(fmt.Sprintf("parse config from config.yaml failed:%s", err))
	}
	fmt.Println("-------->nacos conf1:", Cfg.Mysql)
}
