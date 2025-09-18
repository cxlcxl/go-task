package main
package main

import (
	"flag"
	"fmt"
	"os"

	"xxljob-go-executor/migration"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "配置文件路径")
	flag.Parse()

	if configPath == "" {
		fmt.Println("使用方法: migrate -config=config.yaml")
		os.Exit(1)
	}

	fmt.Printf("开始迁移，配置文件: %s\n", configPath)
	migration.RunMigrationCommand(configPath)
}