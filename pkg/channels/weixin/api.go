package weixin

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

const (
	weixinChannelVersion = "2.1.1"
	weixinIlinkAppID     = "bot"
	// 2.1.1 encoded as 0x00MMNNPP => 0x00020101 => 131329
	weixinClientVersion = 131329
)

type ApiClient struct {
	BaseURL    string
	Token      string
	HttpClient *http.Client
}

func NewApiClient(baseURL, token string, proxy string) (*ApiClient, error) {
	if baseURL == "" {
		baseURL = "https://ilinkai.weixin.qq.com/"
	}

	client := &http.Client{
		// Default timeout; will be overridden per context
	}

	if proxy != "" {
		proxyURL, err := url.Parse(proxy)
		if err != nil {
			return nil, fmt.Errorf("invalid proxy URL %q: %w", proxy, err)
		}

		// Clone the default transport so we preserve all default settings (TLS, HTTP/2, timeouts, keep-alives)
		if defaultTransport, ok := http.DefaultTransport.(*http.Transport); ok {
			transport := defaultTransport.Clone()
			transport.Proxy = http.ProxyURL(proxyURL)
			client.Transport = transport
		} else {
			// Fallback: preserve previous behavior if DefaultTransport is not the expected type
			client.Transport = &http.Transport{
				Proxy: http.ProxyURL(proxyURL),
			}
		}
	}

	return &ApiClient{
		BaseURL:    baseURL,
		Token:      token,
		HttpClient: client,
	}, nil
}

func randomWechatUIN() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	uint32Val := binary.BigEndian.Uint32(b[:])
	return base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%d", uint32Val)))
}

func (c *ApiClient) post(ctx context.Context, endpoint string, body any, responseObj any) error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, endpoint)

	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", u.String(), bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header["iLink-App-Id"] = []string{weixinIlinkAppID}
	req.Header["iLink-App-ClientVersion"] = []string{strconv.Itoa(weixinClientVersion)}
	if endpoint != "ilink/bot/get_bot_qrcode" && endpoint != "ilink/bot/get_qrcode_status" {
		req.Header["AuthorizationType"] = []string{"ilink_bot_token"}
		req.Header["X-WECHAT-UIN"] = []string{randomWechatUIN()}
		if c.Token != "" {
			req.Header.Set("Authorization", "Bearer "+c.Token)
		}
	}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http POST %s failed: %w", endpoint, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http %d %s: %s", resp.StatusCode, resp.Status, string(respBody))
	}

	if responseObj != nil {
		if err := json.Unmarshal(respBody, responseObj); err != nil {
			return fmt.Errorf("failed to unmarshal response: %w, body: %s", err, string(respBody))
		}
	}

	return nil
}

func (c *ApiClient) GetUpdates(ctx context.Context, req GetUpdatesReq) (*GetUpdatesResp, error) {
	req.BaseInfo = BaseInfo{ChannelVersion: weixinChannelVersion}
	var resp GetUpdatesResp
	err := c.post(ctx, "ilink/bot/getupdates", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *ApiClient) SendMessage(ctx context.Context, req SendMessageReq) (*SendMessageResp, error) {
	req.BaseInfo = BaseInfo{ChannelVersion: weixinChannelVersion}
	var resp SendMessageResp
	if err := c.post(ctx, "ilink/bot/sendmessage", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *ApiClient) GetUploadUrl(ctx context.Context, req GetUploadUrlReq) (*GetUploadUrlResp, error) {
	req.BaseInfo = BaseInfo{ChannelVersion: weixinChannelVersion}
	var resp GetUploadUrlResp
	err := c.post(ctx, "ilink/bot/getuploadurl", req, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *ApiClient) GetConfig(ctx context.Context, req GetConfigReq) (*GetConfigResp, error) {
	req.BaseInfo = BaseInfo{ChannelVersion: weixinChannelVersion}
	var resp GetConfigResp
	if err := c.post(ctx, "ilink/bot/getconfig", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *ApiClient) SendTyping(ctx context.Context, req SendTypingReq) (*SendTypingResp, error) {
	req.BaseInfo = BaseInfo{ChannelVersion: weixinChannelVersion}
	var resp SendTypingResp
	if err := c.post(ctx, "ilink/bot/sendtyping", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (c *ApiClient) getQR(ctx context.Context, endpoint string, query map[string]string, respObj any) error {
	u, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	u.Path = path.Join(u.Path, endpoint)
	q := u.Query()
	for key, value := range query {
		q.Set(key, value)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return err
	}
	req.Header["iLink-App-Id"] = []string{weixinIlinkAppID}
	req.Header["iLink-App-ClientVersion"] = []string{strconv.Itoa(weixinClientVersion)}

	resp, err := c.HttpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%s failed: %d %s", endpoint, resp.StatusCode, string(respBody))
	}
	if err := json.Unmarshal(respBody, respObj); err != nil {
		return err
	}

	return nil
}

func (c *ApiClient) GetQRCode(ctx context.Context, botType string) (*QRCodeResponse, error) {
	// get_bot_qrcode is GET, not POST
	var qrcodeResp QRCodeResponse
	if err := c.getQR(ctx, "ilink/bot/get_bot_qrcode", map[string]string{
		"bot_type": botType,
	}, &qrcodeResp); err != nil {
		return nil, err
	}
	return &qrcodeResp, nil
}

func (c *ApiClient) GetQRCodeStatus(ctx context.Context, qrcode string) (*StatusResponse, error) {
	// get_qrcode_status is GET
	var statusResp StatusResponse
	if err := c.getQR(ctx, "ilink/bot/get_qrcode_status", map[string]string{
		"qrcode": qrcode,
	}, &statusResp); err != nil {
		return nil, err
	}
	return &statusResp, nil
}
