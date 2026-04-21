package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"grid_trade/internal/config"
	"grid_trade/internal/debughttp"
	"grid_trade/internal/debugview"
	"grid_trade/internal/gateway"
	runtimepkg "grid_trade/internal/strategy/runtime"
)

type app struct {
	gateway  *gateway.Service
	runtime  *runtimepkg.Runtime
	httpAddr string
	handler  *http.ServeMux
}

func BuildApp(cfg config.Config) (*app, error) {
	gw := gateway.NewService(nil, nil)
	rt := runtimepkg.New()
	view := debugview.NewService(gw)
	debugHandler := debughttp.NewHandler(view)
	mux := http.NewServeMux()
	mux.Handle("/debug/dashboard", debugHandler)
	mux.Handle("/debug/accounts", debugHandler)
	mux.Handle("/debug/events", debugHandler)
	fileServer := http.FileServer(http.Dir("web/debug"))
	mux.HandleFunc("/debug/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/debug" || r.URL.Path == "/debug/" {
			http.ServeFile(w, r, "web/debug/index.html")
			return
		}
		if strings.HasPrefix(r.URL.Path, "/debug/static/") {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/debug/static")
			fileServer.ServeHTTP(w, r)
			return
		}
		http.NotFound(w, r)
	})
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(gw.Health()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

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
		handler:  mux,
	}, nil
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	app, err := BuildApp(config.Config{})
	if err != nil {
		log.Fatalf("build app: %v", err)
	}

	go func() {
		if err := http.ListenAndServe(app.httpAddr, app.handler); err != nil && err != http.ErrServerClosed {
			log.Printf("health server stopped: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("trader stopped")
}
