package binance

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"ai-auto-trade/internal/application/trading"
)

// ExchangeAdapter implements the trading.Exchange interface.
type ExchangeAdapter struct {
	client *Client
}

func NewExchangeAdapter(client *Client) *ExchangeAdapter {
	return &ExchangeAdapter{client: client}
}

func (a *ExchangeAdapter) GetBalance(ctx context.Context, asset string) (float64, error) {
	info, err := a.client.GetAccountInfo()
	if err != nil {
		return 0, err
	}
	for _, b := range info.Balances {
		if strings.EqualFold(b.Asset, asset) {
			val, _ := strconv.ParseFloat(b.Free, 64)
			return val, nil
		}
	}
	return 0, nil
}

func (a *ExchangeAdapter) GetOrder(ctx context.Context, symbol, orderID string) (trading.OrderResponse, error) {
	id, _ := strconv.ParseInt(orderID, 10, 64)
	res, err := a.client.GetOrder(symbol, id)
	if err != nil {
		return trading.OrderResponse{}, err
	}
	p, _ := strconv.ParseFloat(res.Price, 64)
	q, _ := strconv.ParseFloat(res.OrigQty, 64)
	return trading.OrderResponse{
		OrderID: strconv.FormatInt(res.OrderID, 10),
		Symbol:  res.Symbol,
		Side:    res.Side,
		Price:   p,
		Qty:     q,
		Status:  res.Status,
	}, nil
}

func (a *ExchangeAdapter) GetPrice(ctx context.Context, symbol string) (float64, error) {
	return a.client.GetPrice(symbol)
}

func (a *ExchangeAdapter) PlaceMarketOrder(ctx context.Context, symbol, side string, qty float64) (float64, error) {
	return a.placeOrder(symbol, side, fmt.Sprintf("%f", qty), "")
}

func (a *ExchangeAdapter) PlaceMarketOrderQuote(ctx context.Context, symbol, side string, quoteAmount float64) (float64, error) {
	return a.placeOrder(symbol, side, "", fmt.Sprintf("%f", quoteAmount))
}

func (a *ExchangeAdapter) placeOrder(symbol, side, qty, quoteQty string) (float64, error) {
	res, err := a.client.CreateOrder(symbol, strings.ToUpper(side), "MARKET", qty, "", quoteQty)
	if err != nil {
		return 0, err
	}
	
	executedQty, _ := strconv.ParseFloat(res.ExecutedQty, 64)
	if executedQty <= 0 {
		return 0, fmt.Errorf("order executed with zero quantity")
	}

	// Calculate average price from fills
	var totalCost float64
	var totalQty float64
	for _, f := range res.Fills {
		p, _ := strconv.ParseFloat(f.Price, 64)
		q, _ := strconv.ParseFloat(f.Qty, 64)
		totalCost += p * q
		totalQty += q
	}

	if totalQty > 0 {
		return totalCost / totalQty, nil
	}

	return 0, fmt.Errorf("could not determine execution price")
}
