package main

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"time"

	"github.com/openclaw/go_node/config"
	"github.com/openclaw/go_node/gateway"
	"github.com/openclaw/go_node/infra"
	"github.com/openclaw/go_node/node"
)

const version = "0.1.0"

func main() {
	configPath := flag.String("config", "", "path to config.json (default: config.json, ~/.openclaw/go_node.json)")
	initConfig := flag.Bool("init-config", false, "write example config.json and exit")
	flag.Parse()

	if *initConfig {
		path := strings.TrimSpace(*configPath)
		if path == "" {
			path = "config.json"
		}
		if err := config.Example(path); err != nil {
			log.Fatalf("init config: %v", err)
		}
		log.Printf("wrote example config to %s", path)
		return
	}

	cfg, err := config.Load(strings.TrimSpace(*configPath))
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	identity, err := identityFromConfig(cfg)
	if err != nil {
		log.Fatalf("load device identity: %v", err)
	}

	invokeDispatcher := node.NewInvokeDispatcher(cfg.Exec)
	opts := gateway.ConnectOptions{
		Client: gateway.ClientInfo{
			ID:       "node-host",
			Version:  version,
			Platform: runtime.GOOS,
			Mode:     "node",
		},
		Role:      "node",
		Scopes:    []string{},
		Caps:      []string{"system", "file", "camera"},
		Commands:  []string{"system.run", "file.save", "camera.snap"},
		Locale:    "en-US",
		UserAgent: "OpenClawGoNode/" + version + " (" + runtime.GOOS + "; " + runtime.Version() + ")",
	}
	if cfg.Node.DisplayName != "" {
		opts.Client.DisplayName = &cfg.Node.DisplayName
	}
	if cfg.Node.NodeID != "" {
		opts.Client.InstanceID = &cfg.Node.NodeID
	}

	onInvoke := func(req gateway.InvokeRequest) gateway.InvokeResult {
		return invokeDispatcher.HandleInvoke(req)
	}

	sess := gateway.NewSession(
		cfg.WebSocketURL(),
		cfg.Gateway.Token,
		cfg.Gateway.Password,
		opts,
		identity,
		onInvoke,
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		cancel()
	}()

	interval := time.Duration(cfg.Reconnect.RetryIntervalMs) * time.Millisecond
	if interval <= 0 {
		interval = 5 * time.Second
	}
	var attempt int
reconnect:
	for {
		if ctx.Err() != nil {
			break reconnect
		}
		if attempt > 0 {
			log.Printf("go_node: reconnect attempt %d", attempt)
		} else {
			log.Printf("go_node: connecting to %s as node %q", cfg.WebSocketURL(), cfg.Node.DisplayName)
		}
		err := sess.Run(ctx)
		if ctx.Err() != nil {
			break reconnect
		}
		attempt++
		if cfg.Reconnect.MaxRetries > 0 && attempt > cfg.Reconnect.MaxRetries {
			log.Fatalf("go_node: max retries (%d) reached, last error: %v", cfg.Reconnect.MaxRetries, err)
		}
		log.Printf("go_node: connection lost: %v, retrying in %v", err, interval)
		select {
		case <-time.After(interval):
			continue
		case <-ctx.Done():
			break reconnect
		}
	}
	log.Println("go_node: exited")
}

// identityFromConfig builds a DeviceIdentity from cfg.Identity.
// If identity information is missing or invalid, it falls back to generating
// a fresh in-memory identity (not persisted).
func identityFromConfig(cfg *config.Config) (*infra.DeviceIdentity, error) {
	idCfg := cfg.Identity
	if idCfg.DeviceID == "" || idCfg.PublicKeyB64 == "" || idCfg.PrivateKeyB64 == "" {
		return infra.GenerateDeviceIdentity()
	}

	pubRaw, err := base64.StdEncoding.DecodeString(idCfg.PublicKeyB64)
	if err != nil || len(pubRaw) != ed25519.PublicKeySize {
		return infra.GenerateDeviceIdentity()
	}
	privRaw, err := base64.StdEncoding.DecodeString(idCfg.PrivateKeyB64)
	if err != nil || len(privRaw) != ed25519.PrivateKeySize {
		return infra.GenerateDeviceIdentity()
	}

	return &infra.DeviceIdentity{
		DeviceID:     idCfg.DeviceID,
		PublicKeyRaw: pubRaw,
		PrivateKey:   ed25519.PrivateKey(privRaw),
	}, nil
}
