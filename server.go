package main

import (
	"common"
	"flag"
	"fmt"
	"github.com/astaxie/beego/logs"
	_ "net/http/pprof"
	"os"
	"snowflake"
)

func main() {

	// 从命令行获取配置文件路径
	configPathValue := flag.String("config", "config.xml", fmt.Sprintf("%s -config config.xml", os.Args[0]))
	flag.Parse()

	// 加载解析配置
	config, err := common.LoadLeafConfig(*configPathValue)
	if err != nil {
		logs.Error("load config.xml failed, err %v", err)
		return
	}

	err = common.InitLogger(config.LogName, config.LogLevel)
	if err != nil {
		logs.Error(err)
		return
	}

	// 启动snowflake服务
	var snow snowflake.SnowFlake
	if err = snow.Start(&config); err != nil {
		logs.Error("leaf snow flake start failed, err %v", err)
		return
	}
	defer snow.Stop()

	snow.Run(config)

	logs.Info("main over...")
}
