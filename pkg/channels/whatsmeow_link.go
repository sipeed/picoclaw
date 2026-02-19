package channels

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mdp/qrterminal/v3"
	"github.com/skip2/go-qrcode"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "modernc.org/sqlite"
)

// LinkWhatsmeow initiates WhatsApp QR pairing.
// mode can be "terminal" (render QR in terminal) or "png" (save QR as PNG file).
func LinkWhatsmeow(dbPath string, mode string) error {
	dbPath = expandHomePath(dbPath)
	ctx := context.Background()

	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return fmt.Errorf("failed to create db directory: %w", err)
	}

	dbLog := waLog.Noop
	container, err := sqlstore.New(ctx, "sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath), dbLog)
	if err != nil {
		return fmt.Errorf("failed to open whatsmeow db: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device store: %w", err)
	}

	if deviceStore.ID != nil {
		fmt.Printf("Device already linked: %s\n", deviceStore.ID.String())
		fmt.Println("To re-link, delete the database first:")
		fmt.Printf("  rm %s\n", dbPath)
		return nil
	}

	client := whatsmeow.NewClient(deviceStore, waLog.Noop)

	// Listen for Connected event to know when initial sync is done
	connected := make(chan struct{}, 1)
	client.AddEventHandler(func(evt interface{}) {
		switch evt.(type) {
		case *events.Connected:
			select {
			case connected <- struct{}{}:
			default:
			}
		}
	})

	qrChan, err := client.GetQRChannel(ctx)
	if err != nil {
		return fmt.Errorf("failed to get QR channel: %w", err)
	}
	if err := client.Connect(); err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}

	for evt := range qrChan {
		switch evt.Event {
		case "code":
			switch mode {
			case "png":
				pngPath := filepath.Join(filepath.Dir(dbPath), "whatsapp-qr.png")
				if err := qrcode.WriteFile(evt.Code, qrcode.Medium, 256, pngPath); err != nil {
					return fmt.Errorf("failed to write QR PNG: %w", err)
				}
				fmt.Printf("QR code saved to: %s\n", pngPath)
				fmt.Println("Scan the QR code with WhatsApp to link this device.")
			default: // "terminal"
				fmt.Println("Scan this QR code with WhatsApp:")
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			}
		case "success":
			fmt.Println("Paired! Waiting for initial sync...")
			// Wait for the Connected event so the phone finishes the handshake
			select {
			case <-connected:
				fmt.Println("WhatsApp device linked successfully!")
			case <-time.After(30 * time.Second):
				fmt.Println("WhatsApp device paired (sync timed out, but link should work).")
			}
			client.Disconnect()
			return nil
		case "timeout":
			client.Disconnect()
			return fmt.Errorf("QR code scan timed out; run the command again to get a new QR code")
		case "error", "err-unexpected-state", "err-client-outdated", "err-scanned-without-multidevice":
			client.Disconnect()
			return fmt.Errorf("QR pairing failed: %s", evt.Event)
		}
	}

	client.Disconnect()
	return fmt.Errorf("QR channel closed unexpectedly")
}

// WhatsmeowStatus checks if a WhatsApp device is linked in the store.
func WhatsmeowStatus(dbPath string) error {
	dbPath = expandHomePath(dbPath)

	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		fmt.Println("No WhatsApp database found.")
		fmt.Println("Run 'picoclaw whatsapp link' to link a device.")
		return nil
	}

	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite", fmt.Sprintf("file:%s?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)", dbPath), waLog.Noop)
	if err != nil {
		return fmt.Errorf("failed to open db: %w", err)
	}

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		return fmt.Errorf("failed to get device: %w", err)
	}

	if deviceStore.ID == nil {
		fmt.Println("No linked device found.")
		fmt.Println("Run 'picoclaw whatsapp link' to link a device.")
	} else {
		fmt.Printf("Linked device: %s\n", deviceStore.ID.String())
	}

	return nil
}
