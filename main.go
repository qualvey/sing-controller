package main

import (
	"context"
	"embed"
	"flag"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/sagernet/sing-box-webui/internal/api"
	"github.com/sagernet/sing-box-webui/internal/runner"
	"github.com/sagernet/sing-box-webui/internal/store"
)

//go:embed all:web/dist
var webFS embed.FS

func main() {
	var (
		listenAddr  string
		configPath  string
		secret      string
		noRun       bool
	)
	flag.StringVar(&listenAddr, "listen", "127.0.0.1:8080", "HTTP listen address")
	flag.StringVar(&configPath, "config", "config.json", "sing-box config file path")
	flag.StringVar(&secret, "secret", "", "optional API secret (X-Secret header)")
	flag.BoolVar(&noRun, "no-run", false, "config management only, do not start sing-box instance")
	flag.Parse()

	ctx := context.Background()

	// 配置存储（含校验管线）
	cfgStore := store.New(configPath)
	if err := cfgStore.Load(ctx); err != nil {
		log.Fatalf("load config: %v", err)
	}

	// 实例管理（可选运行 sing-box）
	var instanceRunner *runner.Runner
	if !noRun {
		instanceRunner = runner.New()
		if err := instanceRunner.Start(ctx, cfgStore.Options); err != nil {
			log.Printf("WARN: instance start failed: %v (config 仍可管理)", err)
		}
	}

	// 前端静态资源（go:embed）
	dist, err := fs.Sub(webFS, "web/dist")
	if err != nil {
		log.Fatalf("embed dist: %v", err)
	}

	handler := api.NewHandler(api.HandlerOptions{
		Store:   cfgStore,
		Runner:  instanceRunner,
		Secret:  secret,
		Static:  http.FileServer(http.FS(dist)),
		NoRun:   noRun,
	})
	log.Printf("sing-box-webui listening on %s (config: %s)", listenAddr, configPath)
	if err := http.ListenAndServe(listenAddr, handler); err != nil {
		log.Fatal(err)
	}
	_ = os.Stdout
}
