//go:build darwin && !cgo

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
)

func runTray() {
	logger.Infof("System tray is unavailable in darwin builds without cgo; running without tray")

	if !*noBrowser {
		go func() {
			time.Sleep(browserDelay)
			if err := openBrowser(); err != nil {
				logger.Errorf("Warning: Failed to auto-open browser: %v", err)
			}
		}()
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()
	shutdownApp()
}
