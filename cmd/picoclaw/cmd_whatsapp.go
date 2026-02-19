// PicoClaw - Ultra-lightweight personal AI agent
// License: MIT

package main

import (
	"fmt"
	"os"

	"github.com/sipeed/picoclaw/pkg/channels"
)

func whatsappCmd() {
	if len(os.Args) < 3 {
		whatsappHelp()
		return
	}

	cfg, err := loadConfig()
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
		os.Exit(1)
	}

	dbPath := cfg.Channels.Whatsmeow.DBPath

	subcommand := os.Args[2]
	switch subcommand {
	case "link":
		mode := "terminal"
		args := os.Args[3:]
		for i := 0; i < len(args); i++ {
			switch args[i] {
			case "--mode", "-m":
				if i+1 < len(args) {
					mode = args[i+1]
					i++
				}
			}
		}
		if err := channels.LinkWhatsmeow(dbPath, mode); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	case "status":
		if err := channels.WhatsmeowStatus(dbPath); err != nil {
			fmt.Printf("Error: %v\n", err)
			os.Exit(1)
		}
	default:
		fmt.Printf("Unknown whatsapp command: %s\n", subcommand)
		whatsappHelp()
	}
}

func whatsappHelp() {
	fmt.Println("\nWhatsApp commands:")
	fmt.Println("  link        Link a WhatsApp device via QR code")
	fmt.Println("  status      Check WhatsApp device link status")
	fmt.Println()
	fmt.Println("Link options:")
	fmt.Println("  --mode terminal    Render QR in terminal (default)")
	fmt.Println("  --mode png         Save QR as PNG to ~/.picoclaw/whatsapp-qr.png")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  picoclaw whatsapp link")
	fmt.Println("  picoclaw whatsapp link --mode png")
	fmt.Println("  picoclaw whatsapp status")
}
