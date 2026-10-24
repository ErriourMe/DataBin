package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"databin/internal/config"
	"databin/internal/paste"
	"databin/internal/version"
	"databin/internal/web"
)

func main() {
	_ = godotenv.Load()

	cfg := config.Load()

	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis: %v", err)
	}

	tpl, err := web.LoadTemplates(cfg.ViewsDir)
	if err != nil {
		log.Fatalf("templates: %v", err)
	}

	pastes := paste.NewService(rdb, cfg.Paste)
	h := web.NewHandler(cfg, pastes, version.NewService(), tpl)

	mux := http.NewServeMux()
	mux.Handle("GET /{$}", http.HandlerFunc(h.New))
	mux.Handle("POST /documents", http.HandlerFunc(h.Create))
	mux.Handle("PUT /documents/{id}", http.HandlerFunc(h.Update))
	mux.Handle("DELETE /documents/{id}", http.HandlerFunc(h.Destroy))
	mux.Handle("GET /raw/{id}", http.HandlerFunc(h.Raw))
	mux.Handle("GET /download/{id}", http.HandlerFunc(h.Download))
	mux.Handle("GET /{id}", http.HandlerFunc(h.Show))
	if info, err := os.Stat(cfg.PublicDir); err == nil && info.IsDir() {
		mux.Handle("GET /public/", http.StripPrefix("/public/", http.FileServer(http.Dir(cfg.PublicDir))))
	}

	addr := fmt.Sprintf("%s:%d", cfg.AppHost, cfg.AppPort)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	go func() {
		log.Printf("DataBin listening on http://%s", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	_ = server.Shutdown(shutdownCtx)
	_ = rdb.Close()
}
