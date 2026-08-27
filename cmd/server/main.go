package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"textilepermit/internal/httpui"
	"textilepermit/internal/store"
	"textilepermit/internal/workflow"
	"time"
)

func main() {
	cfg := config{}
	flag.StringVar(&cfg.addr, "addr", defaultAddress(), "HTTP 监听地址")
	flag.StringVar(&cfg.dbPath, "db", "textile-permits.db", "SQLite 数据库路径")
	flag.BoolVar(&cfg.selfcheck, "selfcheck", false, "运行真实 HTTP 端到端自检后退出")
	flag.Parse()
	if err := run(cfg); err != nil {
		slog.Error("服务退出", "error", err)
		os.Exit(1)
	}
}
func run(cfg config) error {
	if err := validateAddress(cfg.addr); err != nil {
		return err
	}
	if cfg.selfcheck {
		f, err := os.CreateTemp("", "textile-permit-*.db")
		if err != nil {
			return err
		}
		cfg.dbPath = f.Name()
		f.Close()
		defer os.Remove(cfg.dbPath)
	}
	repo, err := store.Open(cfg.dbPath)
	if err != nil {
		return fmt.Errorf("打开数据库: %w", err)
	}
	defer repo.Close()
	app := workflow.New(repo)
	handler := httpui.New(app, cfg.selfcheck).Handler()
	listener, err := net.Listen("tcp", cfg.addr)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", cfg.addr, err)
	}
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	serveErr := make(chan error, 1)
	go func() {
		e := srv.Serve(listener)
		if e != nil && e != http.ErrServerClosed {
			serveErr <- e
		}
		close(serveErr)
	}()
	slog.Info("光照放行台已启动", "addr", listener.Addr().String())
	if cfg.selfcheck {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		err = runSelfcheck(ctx, "http://"+listener.Addr().String())
		shutdownCtx, stop := context.WithTimeout(context.Background(), 2*time.Second)
		defer stop()
		_ = srv.Shutdown(shutdownCtx)
		if err != nil {
			return err
		}
		slog.Info("端到端自检通过")
		return nil
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
	case err = <-serveErr:
		if err != nil {
			return err
		}
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}
