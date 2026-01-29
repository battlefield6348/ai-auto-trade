package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	apiKey     string
	apiSecret  string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey, apiSecret string, useTestnet bool) *Client {
	baseURL := "https://api.binance.com"
	if useTestnet {
		baseURL = "https://testnet.binance.vision"
	}
	return &Client{
		apiKey:     apiKey,
		apiSecret:  apiSecret,
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) sign(query string) string {
	h := hmac.New(sha256.New, []byte(c.apiSecret))
	h.Write([]byte(query))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Client) call(method, path string, params url.Values, signed bool) ([]byte, error) {
	if signed {
		params.Set("timestamp", fmt.Sprintf("%d", time.Now().UnixMilli()))
		signature := c.sign(params.Encode())
		params.Set("signature", signature)
	}

	fullURL := fmt.Sprintf("%s%s", c.baseURL, path)
	if len(params) > 0 {
		fullURL = fmt.Sprintf("%s?%s", fullURL, params.Encode())
	}

	req, err := http.NewRequest(method, fullURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-MBX-APIKEY", c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("binance api error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

type AccountInfo struct {
	MakerCommission  int `json:"makerCommission"`
	TakerCommission  int `json:"takerCommission"`
	BuyerCommission  int `json:"buyerCommission"`
	SellerCommission int `json:"sellerCommission"`
	CanTrade         bool `json:"canTrade"`
	CanWithdraw      bool `json:"canWithdraw"`
	CanDeposit       bool `json:"canDeposit"`
	UpdateTime       int64 `json:"updateTime"`
	AccountType      string `json:"accountType"`
	Balances         []struct {
		Asset  string `json:"asset"`
		Free   string `json:"free"`
		Locked string `json:"locked"`
	} `json:"balances"`
}

func (c *Client) GetAccountInfo() (*AccountInfo, error) {
	body, err := c.call("GET", "/api/v3/account", url.Values{}, true)
	if err != nil {
		return nil, err
	}
	var info AccountInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

type OrderResponse struct {
	Symbol              string `json:"symbol"`
	OrderID             int64  `json:"orderId"`
	ClientOrderID       string `json:"clientOrderId"`
	TransactTime        int64  `json:"transactTime"`
	Price               string `json:"price"`
	OrigQty             string `json:"origQty"`
	ExecutedQty         string `json:"executedQty"`
	Status              string `json:"status"`
	Type                string `json:"type"`
	Side                string `json:"side"`
	Fills               []struct {
		Price           string `json:"price"`
		Qty             string `json:"qty"`
		Commission      string `json:"commission"`
		CommissionAsset string `json:"commissionAsset"`
	} `json:"fills"`
}

func (c *Client) CreateOrder(symbol, side, orderType, quantity string, price string, quoteQty string) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("side", side)
	params.Set("type", orderType)
	if quantity != "" {
		params.Set("quantity", quantity)
	}
	if quoteQty != "" {
		params.Set("quoteOrderQty", quoteQty)
	}
	if price != "" {
		params.Set("price", price)
		params.Set("timeInForce", "GTC")
	}

	body, err := c.call("POST", "/api/v3/order", params, true)
	if err != nil {
		return nil, err
	}
	var res OrderResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) GetOrder(symbol string, orderID int64) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", fmt.Sprintf("%d", orderID))

	body, err := c.call("GET", "/api/v3/order", params, true)
	if err != nil {
		return nil, err
	}
	var res OrderResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

func (c *Client) CancelOrder(symbol string, orderID int64) (*OrderResponse, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	params.Set("orderId", fmt.Sprintf("%d", orderID))

	body, err := c.call("DELETE", "/api/v3/order", params, true)
	if err != nil {
		return nil, err
	}
	var res OrderResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

type PriceTicker struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
}

func (c *Client) GetPrice(symbol string) (float64, error) {
	params := url.Values{}
	params.Set("symbol", symbol)
	body, err := c.call("GET", "/api/v3/ticker/price", params, false)
	if err != nil {
		return 0, err
	}
	var ticker PriceTicker
	if err := json.Unmarshal(body, &ticker); err != nil {
		return 0, err
	}
	var p float64
	fmt.Sscanf(ticker.Price, "%f", &p)
	return p, nil
}
