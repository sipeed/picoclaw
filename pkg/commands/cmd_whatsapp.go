package commands

import (
	"context"
	"fmt"
)

type qrProvider interface {
	GetLastQR() string
}

func whatsappCommand() Definition {
	return Definition{
		Name:        "whatsapp",
		Description: "WhatsApp management commands",
		SubCommands: []SubCommand{
			{
				Name:        "qr",
				Description: "Get the latest WhatsApp pairing QR code",
				Handler: func(ctx context.Context, req Request, rt *Runtime) error {
					ch, ok := rt.GetChannel("whatsapp_native")
					if !ok {
						return req.Reply("whatsapp_native channel is not enabled")
					}

					qp, ok := ch.(qrProvider)
					if !ok {
						return req.Reply("whatsapp_native channel does not support QR retrieval")
					}

					qr := qp.GetLastQR()
					if qr == "" {
						return req.Reply("no QR code available yet. please wait for the channel to initialize")
					}

					return req.Reply(fmt.Sprintf("Scan this QR code string (or wait for image support): %s", qr))
				},
			},
		},
	}
}
