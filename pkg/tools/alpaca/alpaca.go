package alpaca

import (
	"context"
	"fmt"

	"github.com/alpacahq/alpaca-trade-api-go/v3/alpaca"
	"github.com/alpacahq/alpaca-trade-api-go/v3/marketdata"
	"jane/pkg/tools"
)

type AlpacaTool struct {
	client     *alpaca.Client
	marketData *marketdata.Client
}

func NewAlpacaTool(keyID, secretKey, baseURL string) *AlpacaTool {
	client := alpaca.NewClient(alpaca.ClientOpts{
		APIKey:    keyID,
		APISecret: secretKey,
		BaseURL:   baseURL,
	})
	marketData := marketdata.NewClient(marketdata.ClientOpts{
		APIKey:    keyID,
		APISecret: secretKey,
	})
	return &AlpacaTool{
		client:     client,
		marketData: marketData,
	}
}

func (t *AlpacaTool) Name() string {
	return "alpaca_finance"
}

func (t *AlpacaTool) Description() string {
	return "Provides financial market data and account information via Alpaca API."
}

func (t *AlpacaTool) Parameters() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"action": map[string]any{
				"type":        "string",
				"description": "The action to perform: 'equity', 'price', or 'sma'.",
				"enum":        []string{"equity", "price", "sma"},
			},
			"symbol": map[string]any{
				"type":        "string",
				"description": "Stock symbol (e.g., AAPL). Required for 'price' and 'sma'.",
			},
		},
		"required": []string{"action"},
	}
}

func (t *AlpacaTool) Execute(ctx context.Context, args map[string]any) *tools.ToolResult {
	action, ok := args["action"].(string)
	if !ok {
		return tools.ErrorResult("missing or invalid 'action' parameter")
	}

	if !ok {
		return tools.ErrorResult("missing or invalid 'action' parameter")
	}

	switch action {
	case "equity":
		return t.getEquity()
	case "price":
		symbol, ok := args["symbol"].(string)
		if !ok || symbol == "" {
			return tools.ErrorResult("missing or invalid 'symbol' parameter for price action")
		}
		return t.getPrice(symbol)
	case "sma":
		symbol, ok := args["symbol"].(string)
		if !ok || symbol == "" {
			return tools.ErrorResult("missing or invalid 'symbol' parameter for sma action")
		}
		return t.getSMA(symbol)
	default:
		return tools.ErrorResult(fmt.Sprintf("unknown action: %s", action))
	}
}

func (t *AlpacaTool) getEquity() *tools.ToolResult {
	acct, err := t.client.GetAccount()
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to get account: %v", err))
	}
	return tools.UserResult(fmt.Sprintf("Account Equity: $%s", acct.Equity.String()))
}

func (t *AlpacaTool) getPrice(symbol string) *tools.ToolResult {
	req := marketdata.GetLatestTradeRequest{
		Feed: marketdata.IEX,
	}
	trade, err := t.marketData.GetLatestTrade(symbol, req)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to get latest trade for %s: %v", symbol, err))
	}
	return tools.UserResult(fmt.Sprintf("Latest price for %s: $%.2f", symbol, trade.Price))
}

func (t *AlpacaTool) getSMA(symbol string) *tools.ToolResult {
	req := marketdata.GetBarsRequest{
		TimeFrame:  marketdata.OneDay,
		TotalLimit: 10, // 10-day simple moving average
	}
	bars, err := t.marketData.GetBars(symbol, req)
	if err != nil {
		return tools.ErrorResult(fmt.Sprintf("failed to get bars for %s: %v", symbol, err))
	}
	if len(bars) == 0 {
		return tools.ErrorResult(fmt.Sprintf("no data found for %s", symbol))
	}
	var sum float64
	for _, bar := range bars {
		sum += bar.Close
	}
	sma := sum / float64(len(bars))
	return tools.UserResult(fmt.Sprintf("10-Day SMA for %s: $%.2f", symbol, sma))
}
func init() {
	// tools.Register(&AlpacaTool{}) // We will register it manually where we have access to config.
}

func (t *AlpacaTool) RequiresApproval() bool {
	return false
}
