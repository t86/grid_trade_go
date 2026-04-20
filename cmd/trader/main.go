package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"grid_trade/internal/config"
	"grid_trade/internal/gateway"
	runtimepkg "grid_trade/internal/strategy/runtime"
)

type app struct {
	gateway  *gateway.Service
	runtime  *runtimepkg.Runtime
	httpAddr string
}

func BuildApp(cfg config.Config) (*app, error) {
	gw := gateway.NewService(nil, nil)
	rt := runtimepkg.New()

	addr := cfg.System.HTTPAddr
	if addr == "" {
		addr = ":8080"
	}
	if override := os.Getenv("TRADER_HTTP_ADDR"); override != "" {
		addr = override
	}

	return &app{
		gateway:  gw,
		runtime:  rt,
		httpAddr: addr,
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := BuildApp(config.Config{})
	if err != nil {
		log.Fatalf("build app: %v", err)
	}

	http.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(app.gateway.Health()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	go func() {
		if err := http.ListenAndServe(app.httpAddr, nil); err != nil && err != http.ErrServerClosed {
			log.Printf("health server stopped: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("trader stopped")
}
