// Jane AI Web Console - Web-based chat and management interface
//
// Provides a web UI for chatting with Jane AI via the Pico Channel WebSocket,
// with configuration management and gateway process control.
//
// Usage:
//
//	go build -o jane-ai-web ./web/backend/
//	./jane-ai-web [config.json]
//	./jane-ai-web -public config.json

package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"jane/pkg/logger"
	"jane/web/backend/api"
	"jane/web/backend/launcherconfig"
	"jane/web/backend/middleware"
	"jane/web/backend/utils"
)

func main() {
	port := flag.String("port", "18800", "Port to listen on")
	public := flag.Bool("public", false, "Listen on all interfaces (0.0.0.0) instead of localhost only")
	noBrowser := flag.Bool("no-browser", false, "Do not auto-open browser on startup")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Jane AI Launcher - A web-based configuration editor\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [config.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  config.json    Path to the configuration file (default: ~/.jane-ai/config.json)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s                          Use default config path\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s ./config.json             Specify a config file\n", os.Args[0])
		fmt.Fprintf(
			os.Stderr,
			"  %s -public ./config.json     Allow access from other devices on the network\n",
			os.Args[0],
		)
	}
	flag.Parse()

	// Resolve config path
	configPath := utils.GetDefaultConfigPath()
	if flag.NArg() > 0 {
		configPath = flag.Arg(0)
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		logger.FatalCF("main", "Failed to resolve config path", map[string]any{"error": err.Error()})
	}
	err = utils.EnsureOnboarded(absPath)
	if err != nil {
		logger.WarnCF("main", "Failed to initialize Jane AI config automatically", map[string]any{"error": err.Error()})
	}

	var explicitPort bool
	var explicitPublic bool
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "port":
			explicitPort = true
		case "public":
			explicitPublic = true
		}
	})

	launcherPath := launcherconfig.PathForAppConfig(absPath)
	launcherCfg, err := launcherconfig.Load(launcherPath, launcherconfig.Default())
	if err != nil {
		logger.WarnCF("main", "Failed to load launcher config", map[string]any{"path": launcherPath, "error": err.Error()})
		launcherCfg = launcherconfig.Default()
	}

	effectivePort := *port
	effectivePublic := *public
	if !explicitPort {
		effectivePort = strconv.Itoa(launcherCfg.Port)
	}
	if !explicitPublic {
		effectivePublic = launcherCfg.Public
	}

	portNum, err := strconv.Atoi(effectivePort)
	if err != nil || portNum < 1 || portNum > 65535 {
		if err == nil {
			err = errors.New("must be in range 1-65535")
		}
		logger.FatalCF("main", "Invalid port", map[string]any{"port": effectivePort, "error": err.Error()})
	}

	// Determine listen address
	var addr string
	if effectivePublic {
		addr = "0.0.0.0:" + effectivePort
	} else {
		addr = "127.0.0.1:" + effectivePort
	}

	// Initialize Server components
	mux := http.NewServeMux()

	// API Routes (e.g. /api/status)
	apiHandler := api.NewHandler(absPath)
	apiHandler.SetServerOptions(portNum, effectivePublic, explicitPublic, launcherCfg.AllowedCIDRs)
	apiHandler.RegisterRoutes(mux)

	// Frontend Embedded Assets
	registerEmbedRoutes(mux)

	accessControlledMux, err := middleware.IPAllowlist(launcherCfg.AllowedCIDRs, mux)
	if err != nil {
		logger.FatalCF("main", "Invalid allowed CIDR configuration", map[string]any{"error": err.Error()})
	}

	// Apply middleware stack
	handler := middleware.Recoverer(
		middleware.Logger(
			middleware.JSONContentType(accessControlledMux),
		),
	)

	// Print startup banner
	fmt.Print(utils.Banner)
	fmt.Println()
	fmt.Println("  Open the following URL in your browser:")
	fmt.Println()
	fmt.Printf("    >> http://localhost:%s <<\n", effectivePort)
	if effectivePublic {
		if ip := utils.GetLocalIP(); ip != "" {
			fmt.Printf("    >> http://%s:%s <<\n", ip, effectivePort)
		}
	}
	fmt.Println()

	// Auto-open browser
	if !*noBrowser {
		go func() {
			time.Sleep(500 * time.Millisecond)
			url := "http://localhost:" + effectivePort
			if err := utils.OpenBrowser(url); err != nil {
				logger.WarnCF("main", "Failed to auto-open browser", map[string]any{"error": err.Error()})
			}
		}()
	}

	// Auto-start gateway after backend starts listening.
	go func() {
		time.Sleep(1 * time.Second)
		apiHandler.TryAutoStartGateway()
	}()

	// Start the Server
	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.FatalCF("main", "Server failed to start", map[string]any{"error": err.Error()})
	}
}
