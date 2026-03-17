package main

import (
	"context"
	"fmt"
	"time"

	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

const (
	browserDelay    = 500 * time.Millisecond
	shutdownTimeout = 15 * time.Second
)

func shutdownApp() {
	fmt.Println(T(Exiting))

	if apiHandler != nil {
		apiHandler.Shutdown()
	}

	if server != nil {
		server.SetKeepAlivesEnabled(false)

		ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			if err == context.DeadlineExceeded {
				logger.Infof("Server shutdown timeout after %v, forcing close", shutdownTimeout)
			} else {
				logger.Errorf("Server shutdown error: %v", err)
			}
		} else {
			logger.Infof("Server shutdown completed successfully")
		}
	}
}

func openBrowser() error {
	if serverAddr == "" {
		return fmt.Errorf("server address not set")
	}
	return utils.OpenBrowser(serverAddr)
}
