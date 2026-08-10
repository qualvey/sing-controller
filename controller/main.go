// sing-box-controller：sing-box 配置管理服务（不运行 sing-box 实例）。
// 职责：读取/校验/生成 sing-box 主配置文件，通过 RESTful API 提供给 webui。
// 自身配置 config.json：{"config": "<主配置路径>", "min_port": 8000, "defaults": {...}}
package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/sagernet/sing-box-webui/controller/internal/api"
	"github.com/sagernet/sing-box-webui/controller/internal/settings"
	"github.com/sagernet/sing-box-webui/controller/internal/store"
)

func main() {
	var (
		listenAddr   string
		settingsPath string
		secret       string
	)
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:8080", "HTTP listen address")
	flag.StringVar(&settingsPath, "config", "config.json", "controller config file path")
	flag.StringVar(&secret, "secret", "", "optional API secret (X-Secret header)")
	flag.Parse()

	ctx := context.Background()

	// controller 自身配置
	cfg := settings.New(settingsPath)
	if err := cfg.Load(); err != nil {
		log.Fatalf("load controller config: %v", err)
	}

	// sing-box 主配置存储（路径来自 settings.config）
	values := cfg.Values()
	cfgStore := store.New(values.Config)
	defaults := store.DefaultConfig{
		InboundType: values.Defaults.InboundType,
		Listen:      values.Defaults.Listen,
		ListenPort:  values.Defaults.ListenPort,
	}
	if err := cfgStore.Load(ctx, defaults); err != nil {
		log.Fatalf("load sing-box config: %v", err)
	}

	handler := api.NewHandler(api.HandlerOptions{
		Store:   cfgStore,
		Settings: cfg,
		Secret:  secret,
	})
	log.Printf("sing-box-controller listening on %s (controller config: %s, sing-box config: %s)",
		listenAddr, settingsPath, cfgStore.Path())
	if err := http.ListenAndServe(listenAddr, handler); err != nil {
		log.Fatal(err)
	}
}
