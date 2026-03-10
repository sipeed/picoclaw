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
		Subcommands: []Definition{
			{
				Name:        "qr",
				Description: "Get the latest WhatsApp pairing QR code",
				Handler: func(ctx context.Context, req Request, rt Runtime) ExecuteResult {
					ch, ok := rt.GetChannel("whatsapp_native")
					if !ok {
						return ExecuteResult{
							Outcome: OutcomeHandled,
							Err:     fmt.Errorf("whatsapp_native channel is not enabled"),
						}
					}

					qp, ok := ch.(qrProvider)
					if !ok {
						return ExecuteResult{
							Outcome: OutcomeHandled,
							Err:     fmt.Errorf("whatsapp_native channel does not support QR retrieval"),
						}
					}

					qr := qp.GetLastQR()
					if qr == "" {
						return ExecuteResult{
							Outcome: OutcomeHandled,
							Err:     fmt.Errorf("no QR code available yet. please wait for the channel to initialize"),
						}
					}

					// For now, return the QR code string.
					// Optimization: In the future, we can return an image reference.
					_ = req.Reply(fmt.Sprintf("Scan this QR code string (or wait for image support): %s", qr))

					return ExecuteResult{Outcome: OutcomeHandled}
				},
			},
		},
	}
}
