// sing-box-controllerï¼šsing-box é…ç½®ç®¡ç†æœåŠ¡ï¼ˆä¸è¿è¡Œ sing-box å®žä¾‹ï¼‰ã€‚
// èŒè´£ï¼šè¯»å–/æ ¡éªŒ/ç”Ÿæˆ sing-box ä¸»é…ç½®æ–‡ä»¶ï¼Œé€šè¿‡ RESTful API æä¾›ç»™ webuiã€‚
// è‡ªèº«é…ç½® config.jsonï¼š{"config": "<ä¸»é…ç½®è·¯å¾„>", "min_port": 8000, "defaults": {...}}
package main

import (
	"context"
	"flag"
	"log"
	"net/http"

	"github.com/qualvey/sing-controller/internal/api"
	"github.com/qualvey/sing-controller/internal/settings"
	"github.com/qualvey/sing-controller/internal/store"
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

	// controller è‡ªèº«é…ç½®
	cfg := settings.New(settingsPath)
	if err := cfg.Load(); err != nil {
		log.Fatalf("load controller config: %v", err)
	}

	// sing-box ä¸»é…ç½®å­˜å‚¨ï¼ˆè·¯å¾„æ¥è‡ª settings.configï¼‰
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
