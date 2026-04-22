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
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
	"github.com/sipeed/picoclaw/pkg/netbind"
	"github.com/sipeed/picoclaw/pkg/tools"
	"github.com/sipeed/picoclaw/web/backend/api"
	"github.com/sipeed/picoclaw/web/backend/dashboardauth"
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

	servers    []*http.Server
	serverAddr string
	// browserLaunchURL is opened by openBrowser() (auto-open + tray "open console").
	browserLaunchURL string
	apiHandler       *api.Handler

	noBrowser *bool
)

func shouldEnableLauncherFileLogging(enableConsole, debug bool) bool {
	return !enableConsole || debug
}

func shouldEnableLocalAutoLogin(noBrowser bool, probeHost string) bool {
	return !noBrowser && isLoopbackLaunchHost(probeHost)
}

func isLoopbackLaunchHost(host string) bool {
	host = strings.TrimSpace(host)
	if strings.EqualFold(host, "localhost") {
		return true
	}
	host = strings.Trim(host, "[]")
	if i := strings.LastIndex(host, "%"); i >= 0 {
		host = host[:i]
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func launcherBrowserLaunchSuffix(
	needsSetup bool,
	localAutoLogin *middleware.LauncherDashboardLocalAutoLogin,
) string {
	if needsSetup {
		return middleware.LauncherDashboardSetupPath
	}
	if localAutoLogin != nil {
		return localAutoLogin.URLPath()
	}
	return ""
}

func resolveLauncherHostInput(flagHost string, explicitFlag bool, envHost string) (string, bool, error) {
	if explicitFlag {
		normalized, err := netbind.NormalizeHostInput(flagHost)
		if err != nil {
			return "", false, err
		}
		return normalized, true, nil
	}

	envHost = strings.TrimSpace(envHost)
	if envHost == "" {
		return "", false, nil
	}

	normalized, err := netbind.NormalizeHostInput(envHost)
	if err != nil {
		return "", false, err
	}
	return normalized, true, nil
}

func openLauncherListeners(hostInput string, public bool, port string) (netbind.OpenResult, error) {
	defaultMode := netbind.DefaultLoopback
	if strings.TrimSpace(hostInput) == "" && public {
		defaultMode = netbind.DefaultAny
	}

	plan, err := netbind.BuildPlan(hostInput, defaultMode)
	if err != nil {
		return netbind.OpenResult{}, err
	}
	return netbind.OpenPlan(plan, port)
}

func appendUniqueHost(hosts []string, seen map[string]struct{}, host string) []string {
	host = strings.TrimSpace(host)
	if host == "" {
		return hosts
	}
	key := strings.ToLower(host)
	if _, ok := seen[key]; ok {
		return hosts
	}
	seen[key] = struct{}{}
	return append(hosts, host)
}

func hasWildcardBindHosts(bindHosts []string) bool {
	for _, bindHost := range bindHosts {
		if netbind.IsUnspecifiedHost(bindHost) {
			return true
		}
	}
	return false
}

func wildcardBindHostFamilies(bindHosts []string) (hasIPv4, hasIPv6 bool) {
	for _, bindHost := range bindHosts {
		host := strings.TrimSpace(bindHost)
		if host == "" {
			continue
		}

		if !netbind.IsUnspecifiedHost(host) {
			continue
		}

		ip := net.ParseIP(strings.Trim(host, "[]"))
		if ip == nil {
			continue
		}
		if ip.To4() != nil {
			hasIPv4 = true
			continue
		}
		hasIPv6 = true
	}

	return hasIPv4, hasIPv6
}

func wildcardAdvertiseIP(bindHosts []string, ipv4, ipv6 string) string {
	hasIPv4Wildcard, hasIPv6Wildcard := wildcardBindHostFamilies(bindHosts)
	v4 := strings.TrimSpace(ipv4)
	v6 := strings.TrimSpace(ipv6)

	switch {
	case hasIPv4Wildcard && hasIPv6Wildcard:
		if v6 != "" {
			return v6
		}
		return v4
	case hasIPv6Wildcard:
		return v6
	case hasIPv4Wildcard:
		return v4
	default:
		return ""
	}
}

func advertiseIPForWildcardBindHosts(bindHosts []string) string {
	return wildcardAdvertiseIP(bindHosts, utils.GetLocalIPv4(), utils.GetLocalIPv6())
}

func appendLauncherConsoleHostList(hosts []string, seen map[string]struct{}, values []string) []string {
	for _, value := range values {
		hosts = appendUniqueHost(hosts, seen, value)
	}
	return hosts
}

func shouldShowLocalhostConsoleEntry(hostInput string) bool {
	normalizedHostInput := strings.TrimSpace(hostInput)
	if normalizedHostInput == "" {
		return true
	}

	for token := range strings.SplitSeq(normalizedHostInput, ",") {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if token == "*" || strings.EqualFold(token, "localhost") {
			return true
		}

		ip := net.ParseIP(strings.Trim(token, "[]"))
		if ip == nil {
			continue
		}
		if ip4 := ip.To4(); ip4 != nil {
			if ip4.String() == "127.0.0.1" || ip4.String() == "0.0.0.0" {
				return true
			}
			continue
		}
		if ip.String() == "::1" || ip.String() == "::" {
			return true
		}
	}

	return false
}

func isConsoleDisplayGlobalIPv6(ip net.IP) bool {
	if ip == nil || ip.IsLoopback() || ip.To4() != nil {
		return false
	}
	ip = ip.To16()
	if ip == nil {
		return false
	}
	return ip[0]&0xe0 == 0x20
}

func launcherConsoleHostsWithLocalAddrs(
	hostInput string,
	public bool,
	ipv4s []string,
	globalIPv6s []string,
) []string {
	hosts := make([]string, 0, 8)
	seen := make(map[string]struct{}, 8)

	if shouldShowLocalhostConsoleEntry(hostInput) {
		hosts = appendUniqueHost(hosts, seen, "localhost")
	}

	normalizedHostInput := strings.TrimSpace(hostInput)
	if normalizedHostInput == "" {
		if public {
			hosts = appendLauncherConsoleHostList(hosts, seen, globalIPv6s)
			hosts = appendLauncherConsoleHostList(hosts, seen, ipv4s)
		}
		return hosts
	}

	hasStar := false
	hasIPv4Any := false
	hasIPv6Any := false
	for _, token := range strings.Split(normalizedHostInput, ",") {
		switch strings.TrimSpace(token) {
		case "*":
			hasStar = true
		case "0.0.0.0":
			hasIPv4Any = true
		case "::":
			hasIPv6Any = true
		}
	}

	if hasStar {
		hosts = appendLauncherConsoleHostList(hosts, seen, globalIPv6s)
		hosts = appendLauncherConsoleHostList(hosts, seen, ipv4s)
		return hosts
	}

	for _, token := range strings.Split(normalizedHostInput, ",") {
		token = strings.TrimSpace(token)
		if token == "" || strings.EqualFold(token, "localhost") || netbind.IsLoopbackHost(token) {
			continue
		}

		ip := net.ParseIP(strings.Trim(token, "[]"))
		switch {
		case token == "::":
			hosts = appendLauncherConsoleHostList(hosts, seen, globalIPv6s)
		case token == "0.0.0.0":
			hosts = appendLauncherConsoleHostList(hosts, seen, ipv4s)
		case ip != nil && ip.To4() != nil:
			if hasIPv4Any {
				continue
			}
			hosts = appendUniqueHost(hosts, seen, ip.String())
		case ip != nil:
			if hasIPv6Any {
				continue
			}
			if isConsoleDisplayGlobalIPv6(ip) {
				hosts = appendUniqueHost(hosts, seen, ip.String())
			}
		default:
			hosts = appendUniqueHost(hosts, seen, token)
		}
	}

	return hosts
}

func launcherConsoleHosts(hostInput string, public bool) []string {
	return launcherConsoleHostsWithLocalAddrs(
		hostInput,
		public,
		utils.GetLocalIPv4s(),
		utils.GetGlobalIPv6s(),
	)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func main() {
	port := flag.String("port", "18800", "Port to listen on")
	host := flag.String("host", "", "Host to listen on (overrides -public when set)")
	public := flag.Bool("public", false, "Listen on all interfaces (dual-stack) instead of localhost only")
	noBrowser = flag.Bool("no-browser", false, "Do not auto-open browser on startup")
	lang := flag.String("lang", "", "Language: en (English) or zh (Chinese). Default: auto-detect from system locale")
	console := flag.Bool("console", false, "Console mode, no GUI")

	var debug bool
	flag.BoolVar(&debug, "d", false, "Enable debug logging")
	flag.BoolVar(&debug, "debug", false, "Enable debug logging")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "%s Launcher - Web console and gateway manager\n\n", appName)
		fmt.Fprintf(os.Stderr, "Usage: %s [options] [config.json]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  config.json    Path to the configuration file (default: ~/.picoclaw/config.json)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      Use default config path in GUI mode\n")
		fmt.Fprintf(os.Stderr, "  %s ./config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      Specify a config file\n")
		fmt.Fprintf(
			os.Stderr,
			"  %s -public ./config.json\n",
			os.Args[0],
		)
		fmt.Fprintf(os.Stderr, "      Allow access from other devices on the local network\n")
		fmt.Fprintf(os.Stderr, "  %s -host :: ./config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      Bind launcher host explicitly with exact host semantics\n")
		fmt.Fprintf(os.Stderr, "  %s -console -d ./config.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "      Run in the terminal with debug logs enabled\n")
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

	enableConsole := *console
	fileLoggingEnabled := shouldEnableLauncherFileLogging(enableConsole, debug)
	if fileLoggingEnabled {
		// GUI mode writes launcher logs to file. Debug mode keeps file logging enabled in console mode too.
		if !debug {
			logger.DisableConsole()
		}

		f := filepath.Join(picoHome, logPath, logFile)
		if err = logger.EnableFileLogging(f); err != nil {
			panic(fmt.Sprintf("error enabling file logging: %v", err))
		}
		defer logger.DisableFileLogging()
	}
	if debug {
		logger.SetLevel(logger.DEBUG)
	}

	// Set language from command line or auto-detect
	if *lang != "" {
		SetLanguage(*lang)
	}
	tools.SetPreferredWebSearchLanguage(string(GetLanguage()))

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
		logger.Errorf("Warning: Failed to initialize %s config automatically: %v", appName, err)
	}
	if !debug {
		logger.SetLevelFromString(config.ResolveGatewayLogLevel(absPath))
	}

	logger.InfoC("web", fmt.Sprintf("%s launcher starting (version %s)...", appName, appVersion))
	logger.InfoC("web", fmt.Sprintf("%s Home: %s", appName, picoHome))
	if debug {
		logger.InfoC("web", "Debug mode enabled")
		logger.DebugC(
			"web",
			fmt.Sprintf(
				"Launcher flags: console=%t host=%q public=%t no_browser=%t config=%s",
				enableConsole,
				*host,
				*public,
				*noBrowser,
				absPath,
			),
		)
	}

	var explicitPort bool
	var explicitPublic bool
	var explicitHost bool
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "port":
			explicitPort = true
		case "host":
			explicitHost = true
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
	envHost := strings.TrimSpace(os.Getenv(launcherconfig.EnvLauncherHost))

	hostInput, hostOverrideActive, err := resolveLauncherHostInput(*host, explicitHost, envHost)
	if err != nil {
		logger.Fatalf("Invalid host %q: %v", firstNonEmpty(strings.TrimSpace(*host), envHost), err)
	}
	if hostOverrideActive {
		effectivePublic = false
	}

	if !explicitHost && hostOverrideActive {
		logger.InfoC("web", "Using launcher host from environment PICOCLAW_LAUNCHER_HOST")
	}

	if hostOverrideActive && explicitPublic {
		logger.InfoC("web", "Ignoring -public because launcher host was explicitly set")
	}

	portNum, err := strconv.Atoi(effectivePort)
	if err != nil || portNum < 1 || portNum > 65535 {
		if err == nil {
			err = errors.New("must be in range 1-65535")
		}
		logger.Fatalf("Invalid port %q: %v", effectivePort, err)
	}

	openResult, err := openLauncherListeners(hostInput, effectivePublic, effectivePort)
	if err != nil {
		logger.Fatalf("Failed to open launcher listener(s): %v", err)
	}
	listeners := openResult.Listeners

	dashboardSessionCookie, dashErr := middleware.NewLauncherDashboardSessionCookie()
	if dashErr != nil {
		logger.Fatalf("Dashboard auth setup failed: %v", dashErr)
	}

	// Open the bcrypt password store (creates the DB file on first run).
	authStore, authStoreErr := dashboardauth.New(picoHome)
	var passwordStore api.PasswordStore
	if authStoreErr == nil {
		passwordStore = authStore
		defer authStore.Close()
	} else if errors.Is(authStoreErr, dashboardauth.ErrUnsupportedPlatform) {
		logger.InfoC(
			"web",
			fmt.Sprintf(
				"Dashboard SQLite password store unavailable on this platform; using launcher-config password storage: %v",
				authStoreErr,
			),
		)
		passwordStore = launcherconfig.NewPasswordStore(launcherPath, launcherCfg)
		authStoreErr = nil
	} else {
		logger.ErrorC("web", fmt.Sprintf("Warning: could not open auth store: %v", authStoreErr))
	}

	migrationResult, migrationErr := launcherconfig.MigrateLegacyLauncherToken(
		context.Background(),
		passwordStore,
		launcherPath,
		launcherCfg,
	)
	if migrationErr != nil {
		logger.Fatalf("Failed to migrate legacy launcher token to password login: %v", migrationErr)
	}
	if migrationResult.Migrated {
		logger.InfoC("web", "Migrated legacy launcher token to dashboard password login")
	}
	if migrationResult.CleanupErr != nil {
		logger.WarnC(
			"web",
			fmt.Sprintf(
				"Legacy launcher token password migration succeeded, but failed to remove launcher_token from %s: %v",
				launcherPath,
				migrationResult.CleanupErr,
			),
		)
	}

	var localAutoLogin *middleware.LauncherDashboardLocalAutoLogin
	needsInitialSetup := false
	if passwordStore != nil {
		initialized, initErr := passwordStore.IsInitialized(context.Background())
		if initErr != nil {
			logger.ErrorC("web", fmt.Sprintf("Warning: could not check dashboard password state: %v", initErr))
		} else if !initialized {
			needsInitialSetup = true
		} else if shouldEnableLocalAutoLogin(*noBrowser, openResult.ProbeHost) {
			localAutoLogin, err = middleware.NewLauncherDashboardLocalAutoLogin(5 * time.Minute)
			if err != nil {
				logger.Fatalf("Failed to create local auto-login grant: %v", err)
			}
		}
	}

	// Initialize Server components
	mux := http.NewServeMux()

	api.RegisterLauncherAuthRoutes(mux, api.LauncherAuthRouteOpts{
		SessionCookie: dashboardSessionCookie,
		PasswordStore: passwordStore,
		StoreError:    authStoreErr,
	})

	// API Routes (e.g. /api/status)
	apiHandler = api.NewHandler(absPath)
	apiHandler.SetDebug(debug)
	if _, err = apiHandler.EnsurePicoChannel(); err != nil {
		logger.ErrorC("web", fmt.Sprintf("Warning: failed to ensure pico channel on startup: %v", err))
	}
	apiHandler.SetServerOptions(portNum, effectivePublic, explicitPublic, launcherCfg.AllowedCIDRs)
	apiHandler.SetServerBindHost(hostInput, hostOverrideActive)
	apiHandler.RegisterRoutes(mux)

	// Frontend Embedded Assets
	registerEmbedRoutes(mux)

	accessControlledMux, err := middleware.IPAllowlist(launcherCfg.AllowedCIDRs, mux)
	if err != nil {
		logger.Fatalf("Invalid allowed CIDR configuration: %v", err)
	}

	dashAuth := middleware.LauncherDashboardAuth(middleware.LauncherDashboardAuthConfig{
		ExpectedCookie: dashboardSessionCookie,
		LocalAutoLogin: localAutoLogin,
	}, accessControlledMux)

	// Apply middleware stack
	handler := middleware.Recoverer(
		middleware.Logger(
			middleware.ReferrerPolicyNoReferrer(
				middleware.JSONContentType(dashAuth),
			),
		),
	)

	// Print startup banner (console mode only).
	if enableConsole || debug {
		consoleHosts := launcherConsoleHosts(hostInput, effectivePublic)

		fmt.Print(utils.Banner)
		fmt.Println()
		if needsInitialSetup {
			if *noBrowser {
				fmt.Println("  First-time setup: open /launcher-setup to create the dashboard password.")
			} else {
				fmt.Println("  Launcher will open /launcher-setup automatically.")
			}
			fmt.Println()
		}
		fmt.Println("  Dashboard address:")
		fmt.Println()
		for _, host := range consoleHosts {
			fmt.Printf("    >> http://%s <<\n", net.JoinHostPort(host, effectivePort))
		}
		fmt.Println()
	}

	// Log startup info to file
	for _, ln := range listeners {
		logger.InfoC("web", fmt.Sprintf("Server will listen on http://%s", ln.Addr().String()))
	}
	if hasWildcardBindHosts(openResult.BindHosts) {
		if ip := advertiseIPForWildcardBindHosts(openResult.BindHosts); ip != "" {
			logger.InfoC("web", fmt.Sprintf("Public access enabled at http://%s", net.JoinHostPort(ip, effectivePort)))
		}
	}

	// Share the local URL with the launcher runtime.
	serverAddr = fmt.Sprintf("http://%s", net.JoinHostPort(openResult.ProbeHost, effectivePort))
	browserLaunchURL = serverAddr + launcherBrowserLaunchSuffix(needsInitialSetup, localAutoLogin)

	// Auto-open browser will be handled by the launcher runtime.

	// Auto-start gateway after backend starts listening.
	go func() {
		time.Sleep(1 * time.Second)
		apiHandler.TryAutoStartGateway()
	}()

	// Start the server(s) in goroutines.
	servers = make([]*http.Server, 0, len(listeners))
	for _, ln := range listeners {
		srv := &http.Server{Handler: handler}
		servers = append(servers, srv)

		go func(s *http.Server, l net.Listener) {
			logger.InfoC("web", fmt.Sprintf("Server listening on %s", l.Addr().String()))
			if serveErr := s.Serve(l); serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				logger.Fatalf("Server failed to start on %s: %v", l.Addr().String(), serveErr)
			}
		}(srv, ln)
	}

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
