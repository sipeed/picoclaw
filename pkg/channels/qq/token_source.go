package qq

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/tencent-connect/botgo/constant"
	"github.com/tencent-connect/botgo/token"
	"golang.org/x/oauth2"
	"golang.org/x/sync/singleflight"
)

const defaultQQTokenRequestTimeout = 30 * time.Second

type qqTokenSource struct {
	appID     string
	appSecret string
	tokenURL  string
	client    *http.Client

	cachedToken atomic.Value
	sg          singleflight.Group
}

type qqTokenResponse struct {
	Code        int             `json:"code"`
	Message     string          `json:"message"`
	AccessToken string          `json:"access_token"`
	ExpiresIn   json.RawMessage `json:"expires_in"`
}

func newQQTokenSource(appID, appSecret string) oauth2.TokenSource {
	return &qqTokenSource{
		appID:     appID,
		appSecret: appSecret,
		tokenURL:  fmt.Sprintf("%s/app/getAppAccessToken", constant.TokenDomain),
		client: &http.Client{
			Timeout: defaultQQTokenRequestTimeout,
		},
	}
}

func (s *qqTokenSource) Token() (*oauth2.Token, error) {
	raw := s.cachedToken.Load()
	if raw != nil {
		if tk, ok := raw.(*oauth2.Token); ok && tk != nil && tk.Valid() {
			return tk, nil
		}
	}

	fresh, err, _ := s.sg.Do("qq_access_token", func() (any, error) {
		return s.getNewToken()
	})
	if err != nil {
		return nil, err
	}

	tk, ok := fresh.(*oauth2.Token)
	if !ok || tk == nil {
		return nil, fmt.Errorf("qq token source returned unexpected token type")
	}
	s.cachedToken.Store(tk)
	return tk, nil
}

func (s *qqTokenSource) getNewToken() (*oauth2.Token, error) {
	payload, err := json.Marshal(map[string]string{
		"appId":        s.appID,
		"clientSecret": s.appSecret,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest(http.MethodPost, s.tokenURL, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	rsp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rsp.Body.Close() }()

	body, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, err
	}

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qq token request failed: status=%d body=%s", rsp.StatusCode, strings.TrimSpace(string(body)))
	}

	parsed := &qqTokenResponse{}
	if err := json.Unmarshal(body, parsed); err != nil {
		return nil, fmt.Errorf("qq token response decode failed: %w", err)
	}

	if parsed.Code != 0 {
		return nil, fmt.Errorf("qq token request failed: %d.%s", parsed.Code, parsed.Message)
	}
	if parsed.AccessToken == "" {
		return nil, fmt.Errorf("qq token request failed: empty access token")
	}

	expiresIn, err := parseQQTokenExpiresIn(parsed.ExpiresIn)
	if err != nil {
		return nil, fmt.Errorf("qq token request failed: %w", err)
	}
	if expiresIn <= 0 {
		return nil, fmt.Errorf("qq token request failed: invalid expires_in=%d", expiresIn)
	}

	return &oauth2.Token{
		AccessToken: parsed.AccessToken,
		TokenType:   token.TypeQQBot,
		Expiry:      time.Now().Add(time.Duration(expiresIn) * time.Second),
		ExpiresIn:   expiresIn,
	}, nil
}

func parseQQTokenExpiresIn(raw json.RawMessage) (int64, error) {
	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		parsed, parseErr := strconv.ParseInt(asString, 10, 64)
		if parseErr != nil {
			return 0, fmt.Errorf("invalid expires_in string %q: %w", asString, parseErr)
		}
		return parsed, nil
	}

	var asInt int64
	if err := json.Unmarshal(raw, &asInt); err == nil {
		return asInt, nil
	}

	return 0, fmt.Errorf("invalid expires_in payload %s", string(raw))
}
