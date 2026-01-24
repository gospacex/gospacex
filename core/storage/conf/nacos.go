package conf

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

func Init(path string) {
	ParseConfig(path)
	//判断是否定义了nacos配置
	if Cfg.Nacos != nil {
		cs := []constant.ServerConfig{
			{
				IpAddr: Cfg.Nacos.Host,
				Port:   uint64(Cfg.Nacos.Port),
			},
		}
		cc := constant.ClientConfig{
			NamespaceId:         Cfg.Nacos.NamespaceId,
			TimeoutMs:           5000,
			NotLoadCacheAtStart: true,
			LogDir:              "tmp/nacos/log",
			CacheDir:            "tmp/nacos/cache",
			LogLevel:            "debug",
		}

		configClient, err := clients.CreateConfigClient(map[string]interface{}{
			"serverConfigs": cs,
			"clientConfig":  cc,
		})
		if err != nil {
			panic(err)
		}
		// 配置监听的参数
		param := vo.ConfigParam{
			DataId: Cfg.Nacos.DataId,
			Group:  Cfg.Nacos.Group,
		}
		// 配置监听回调函数，当配置发生变化时会被触发
		callback := func(namespace, group, dataId, data string) {
			confdata := &data
			fmt.Println("-------------->读取配置信息2:", confdata)
			fmt.Println("配置已更新，新内容如下：")
			fmt.Println(data)
		}
		// 启动配置监听
		err = configClient.ListenConfig(vo.ConfigParam{
			DataId:   param.DataId,
			Group:    param.Group,
			OnChange: callback,
		})
		if err != nil {
			panic(err)
		}
		// 首次获取配置并打印
		content, err := configClient.GetConfig(param)
		if err != nil {
			panic(err)
		}

		err = json.Unmarshal([]byte(content), Cfg)
		if err != nil {
			zap.S().Fatalf("读取nacos配置失败： %s", err.Error())
		}
		fmt.Println("-------------->读取配置信息:", Cfg)
	}
}
