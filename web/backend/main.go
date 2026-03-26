// PicoClaw Web Console - Web-based chat and management interface
//
// Provides a web UI for chatting with PicoClaw via the Pico Channel WebSocket,
// with configuration management and gateway process control.
//
// Usage:
//
//	go build -o picoclaw-web ./web/backend/
//	./picoclaw-web [config.json]
//	./picoclaw-web -public config.json

package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/web/backend/api"
	"github.com/sipeed/picoclaw/web/backend/launcherconfig"
	"github.com/sipeed/picoclaw/web/backend/middleware"
	"github.com/sipeed/picoclaw/web/backend/utils"
)

const (
	appName = "PicoClaw"

	logPath   = "logs"
	panicFile = "launcher_panic.log"
	logFile   = "launcher.log"
)

var (
	appVersion = config.Version

	server     *http.Server
	serverAddr string
	apiHandler *api.Handler

	noBrowser *bool
)

func main() {
	port := flag.String("port", "18800", "Port to listen on")
	public := flag.Bool("public", false, "Listen on all interfaces (0.0.0.0) instead of localhost only")
	noBrowser = flag.Bool("no-browser", false, "Do not auto-open browser on startup")
	lang := flag.String("lang", "", "Language: en (English) or zh (Chinese). Default: auto-detect from system locale")
	console := flag.Bool("console", false, "Console mode, no GUI")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "PicoClaw Launcher - A web-based configuration editor\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [config.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  config.json    Path to the configuration file (default: ~/.picoclaw/config.json)\n\n")
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

	// Initialize logger
	picoHome := utils.GetPicoclawHome()

	f := filepath.Join(picoHome, logPath, panicFile)
	panicFunc, err := logger.InitPanic(f)
	if err != nil {
		panic(fmt.Sprintf("error initializing panic log: %v", err))
	}
	defer panicFunc()

	// By default, detect terminal to decide console log behavior
	// If -console-logs flag is explicitly set, it overrides the detection
	enableConsole := *console
	if !enableConsole {
		// Disable console logging by setting level to Fatal (no output)
		logger.SetConsoleLevel(logger.FATAL)

		f := filepath.Join(picoHome, logPath, logFile)
		if err = logger.EnableFileLogging(f); err != nil {
			panic(fmt.Sprintf("error enabling file logging: %v", err))
		}
		defer logger.DisableFileLogging()
	}

	logger.InfoC("web", fmt.Sprintf("%s Launcher %s starting...", appName, appVersion))
	logger.InfoC("web", fmt.Sprintf("PicoClaw Home: %s", picoHome))

	// Set language from command line or auto-detect
	if *lang != "" {
		SetLanguage(*lang)
	}

	// Resolve config path
	configPath := utils.GetDefaultConfigPath()
	if flag.NArg() > 0 {
		configPath = flag.Arg(0)
	}

	absPath, err := filepath.Abs(configPath)
	if err != nil {
		logger.Fatalf("Failed to resolve config path: %v", err)
	}
	err = utils.EnsureOnboarded(absPath)
	if err != nil {
		logger.Errorf("Warning: Failed to initialize PicoClaw config automatically: %v", err)
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
		logger.ErrorC("web", fmt.Sprintf("Warning: Failed to load %s: %v", launcherPath, err))
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
		logger.Fatalf("Invalid port %q: %v", effectivePort, err)
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
	apiHandler = api.NewHandler(absPath)
	if _, err = apiHandler.EnsurePicoChannel(""); err != nil {
		logger.ErrorC("web", fmt.Sprintf("Warning: failed to ensure pico channel on startup: %v", err))
	}
	apiHandler.SetServerOptions(portNum, effectivePublic, explicitPublic, launcherCfg.AllowedCIDRs)
	apiHandler.RegisterRoutes(mux)

	// Frontend Embedded Assets
	registerEmbedRoutes(mux)

	accessControlledMux, err := middleware.IPAllowlist(launcherCfg.AllowedCIDRs, mux)
	if err != nil {
		logger.Fatalf("Invalid allowed CIDR configuration: %v", err)
	}

	// Apply middleware stack
	handler := middleware.Recoverer(
		middleware.Logger(
			middleware.JSONContentType(accessControlledMux),
		),
	)

	// Print startup banner (only in console mode)
	if enableConsole {
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
	}

	// Log startup info to file
	logger.InfoC("web", fmt.Sprintf("Server will listen on http://localhost:%s", effectivePort))
	if effectivePublic {
		if ip := utils.GetLocalIP(); ip != "" {
			logger.InfoC("web", fmt.Sprintf("Public access enabled at http://%s:%s", ip, effectivePort))
		}
	}

	// Share the local URL with the launcher runtime.
	serverAddr = fmt.Sprintf("http://localhost:%s", effectivePort)

	// Auto-open browser will be handled by the launcher runtime.

	// Auto-start gateway after backend starts listening.
	go func() {
		time.Sleep(1 * time.Second)
		apiHandler.TryAutoStartGateway()
	}()

	// Start the Server in a goroutine
	server = &http.Server{Addr: addr, Handler: handler}
	go func() {
		logger.InfoC("web", fmt.Sprintf("Server listening on %s", addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatalf("Server failed to start: %v", err)
		}
	}()

	defer shutdownApp()

	// Start system tray or run in console mode
	if enableConsole {
		if !*noBrowser {
			// Auto-open browser after systray is ready (if not disabled)
			// Check no-browser flag via environment or pass as parameter if needed
			if err := openBrowser(); err != nil {
				logger.Errorf("Warning: Failed to auto-open browser: %v", err)
			}
		}

		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

		// Main event loop - wait for signals or config changes
		for {
			select {
			case <-sigChan:
				logger.Info("Shutting down...")

				return
			}
		}
	} else {
		// GUI mode: start system tray
		runTray()
	}
}
