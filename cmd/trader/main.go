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
	"time"

	"grid_trade/internal/config"
	"grid_trade/internal/debughttp"
	"grid_trade/internal/debugview"
	"grid_trade/internal/domain"
	binance "grid_trade/internal/exchange/binance/common"
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
	startConfiguredAccounts(gw, cfg)
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

	cfg := config.Config{}
	configPath := os.Getenv("TRADER_CONFIG")
	if configPath == "" {
		configPath = "configs/example.yaml"
	}
	if loaded, err := config.LoadFile(configPath); err == nil {
		cfg = loaded
	} else {
		log.Printf("load config %s: %v", configPath, err)
	}

	app, err := BuildApp(cfg)
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

func startConfiguredAccounts(gw *gateway.Service, cfg config.Config) {
	for _, account := range cfg.Accounts {
		markets := parseMarkets(account.MarketTypes)
		apiKey := os.Getenv(account.APIKeyEnv)
		secretKey := os.Getenv(account.SecretKeyEnv)
		if apiKey == "" || secretKey == "" {
			missing := missingCredentialMessage(account)
			for _, market := range markets {
				gw.MarkMarketDegraded(account.Name, market, missing)
			}
			continue
		}

		restClient := binance.NewRESTClient(apiKey, nil, binance.DefaultEndpoints())
		connector := binance.NewUserStreamConnector(binance.DefaultEndpoints(), nil)
		go func(accountName string, markets []domain.MarketType) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			if err := gw.BootstrapAccountWith(ctx, accountName, markets, restClient, connector); err != nil {
				for _, market := range markets {
					gw.MarkMarketDegraded(accountName, market, err.Error())
				}
			}
		}(account.Name, markets)
	}
}

func parseMarkets(values []string) []domain.MarketType {
	markets := make([]domain.MarketType, 0, len(values))
	for _, value := range values {
		market := domain.MarketType(strings.TrimSpace(value))
		switch market {
		case domain.MarketSpot, domain.MarketFuturesUM:
			markets = append(markets, market)
		}
	}
	return markets
}

func missingCredentialMessage(account config.AccountConfig) string {
	missing := make([]string, 0, 2)
	if os.Getenv(account.APIKeyEnv) == "" {
		missing = append(missing, account.APIKeyEnv)
	}
	if os.Getenv(account.SecretKeyEnv) == "" {
		missing = append(missing, account.SecretKeyEnv)
	}
	return "missing credential env: " + strings.Join(missing, ", ")
}
