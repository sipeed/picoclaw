package zalo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

const (
	apiBase   = "https://openapi.zalo.me/v2.0"
	oauthBase = "https://oauth.zaloapp.com/v4"
)

type ZaloAPI struct {
	appID        string
	appSecret    string
	accessToken  string
	refreshToken string
	httpClient   *http.Client
}

func NewZaloAPI(appID, appSecret, accessToken, refreshToken string) *ZaloAPI {
	return &ZaloAPI{
		appID: appID, appSecret: appSecret,
		accessToken: accessToken, refreshToken: refreshToken,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

func (a *ZaloAPI) SetAccessToken(token string) { a.accessToken = token }

func (a *ZaloAPI) SendTextMessage(recipientID, text string) error {
	return a.oaPost("/oa/message", map[string]interface{}{
		"recipient": map[string]string{"user_id": recipientID},
		"message":   map[string]string{"text": text},
	})
}

func (a *ZaloAPI) SendImageMessage(recipientID, imageURL string) error {
	return a.oaPost("/oa/message", map[string]interface{}{
		"recipient": map[string]string{"user_id": recipientID},
		"message": map[string]interface{}{
			"attachment": map[string]interface{}{
				"type": "template",
				"payload": map[string]interface{}{
					"template_type": "media",
					"elements":      []map[string]interface{}{{"media_type": "image", "url": imageURL}},
				},
			},
		},
	})
}

func (a *ZaloAPI) GetUserProfile(userID string) (*ZaloUserProfile, error) {
	req, err := a.newRequest(http.MethodGet,
		fmt.Sprintf("%s/oa/getprofile?data={\"user_id\":\"%s\"}", apiBase, userID), nil)
	if err != nil {
		return nil, err
	}
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zalo get profile: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		Error   int             `json:"error"`
		Message string          `json:"message"`
		Data    ZaloUserProfile `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, err
	}
	if out.Error != 0 {
		return nil, fmt.Errorf("zalo get profile error %d: %s", out.Error, out.Message)
	}
	return &out.Data, nil
}

func (a *ZaloAPI) AuthorizationURL(redirectURI, state, codeChallenge string) string {
	params := url.Values{
		"app_id": {a.appID}, "redirect_uri": {redirectURI},
		"code_challenge": {codeChallenge}, "state": {state},
		"code_challenge_method": {"S256"},
	}
	return oauthBase + "/permission?" + params.Encode()
}

func (a *ZaloAPI) ExchangeCodeForToken(code, codeVerifier, redirectURI string) (*OAuthTokenResponse, error) {
	resp, err := a.httpClient.PostForm(oauthBase+"/access_token", url.Values{
		"app_id": {a.appID}, "app_secret": {a.appSecret},
		"code": {code}, "code_verifier": {codeVerifier},
		"grant_type": {"authorization_code"}, "redirect_uri": {redirectURI},
	})
	if err != nil {
		return nil, fmt.Errorf("zalo token exchange: %w", err)
	}
	defer resp.Body.Close()
	var tok OAuthTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tok); err != nil {
		return nil, err
	}
	if tok.Error != 0 {
		return nil, fmt.Errorf("zalo token error %d: %s", tok.Error, tok.ErrorDescription)
	}
	return &tok, nil
}

func (a *ZaloAPI) RefreshAccessToken() (string, error) {
	resp, err := a.httpClient.PostForm(oauthBase+"/access_token", url.Values{
		"app_id": {a.appID}, "app_secret": {a.appSecret},
		"refresh_token": {a.refreshToken}, "grant_type": {"refresh_token"},
	})
	if err != nil {
		return "", fmt.Errorf("zalo refresh: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		Error            int    `json:"error"`
		ErrorDescription string `json:"error_description"`
		AccessToken      string `json:"access_token"`
		RefreshToken     string `json:"refresh_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Error != 0 {
		return "", fmt.Errorf("zalo refresh error %d: %s", out.Error, out.ErrorDescription)
	}
	if out.RefreshToken != "" {
		a.refreshToken = out.RefreshToken
	}
	return out.AccessToken, nil
}

func (a *ZaloAPI) GetSocialUserProfile(userAccessToken string) (*ZaloUserProfile, error) {
	req, _ := http.NewRequest(http.MethodGet, "https://graph.zalo.me/v2.0/me?fields=id,name,picture", nil)
	req.Header.Set("access_token", userAccessToken)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zalo social profile: %w", err)
	}
	defer resp.Body.Close()
	var p ZaloUserProfile
	json.NewDecoder(resp.Body).Decode(&p)
	return &p, nil
}

func (a *ZaloAPI) GetFriendList(userAccessToken string, offset, count int) ([]ZaloFriend, error) {
	req, _ := http.NewRequest(http.MethodGet,
		fmt.Sprintf("https://graph.zalo.me/v2.0/me/friends?fields=id,name,picture&offset=%d&limit=%d", offset, count), nil)
	req.Header.Set("access_token", userAccessToken)
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("zalo friend list: %w", err)
	}
	defer resp.Body.Close()
	var out struct {
		Data  []ZaloFriend `json:"data"`
		Error int          `json:"error"`
		Msg   string       `json:"message"`
	}
	json.NewDecoder(resp.Body).Decode(&out)
	if out.Error != 0 {
		return nil, fmt.Errorf("zalo friend list error %d: %s", out.Error, out.Msg)
	}
	return out.Data, nil
}

func (a *ZaloAPI) oaPost(path string, payload interface{}) error {
	data, _ := json.Marshal(payload)
	req, err := a.newRequest(http.MethodPost, apiBase+path, bytes.NewReader(data))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("zalo oa post %s: %w", path, err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var out struct {
		Error   int    `json:"error"`
		Message string `json:"message"`
	}
	json.Unmarshal(body, &out)
	if out.Error != 0 {
		return fmt.Errorf("zalo oa error %d: %s", out.Error, out.Message)
	}
	return nil
}

func (a *ZaloAPI) newRequest(method, rawURL string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, rawURL, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("access_token", a.accessToken)
	return req, nil
}

// ── Structs ──────────────────────────────────────────────────────────────────

type WebhookEvent struct {
	AppID     string        `json:"app_id"`
	EventName string        `json:"event_name"`
	Sender    ZaloSender    `json:"sender"`
	Recipient ZaloRecipient `json:"recipient"`
	Message   ZaloMessage   `json:"message"`
	Timestamp int64         `json:"timestamp"`
}
type ZaloSender    struct{ ID string `json:"id"`; Name string `json:"display_name"` }
type ZaloRecipient struct{ ID string `json:"id"` }
type ZaloMessage   struct {
	MsgID       string           `json:"msg_id"`
	Text        string           `json:"text"`
	Attachments []ZaloAttachment `json:"attachments"`
}
type ZaloAttachment struct {
	Type    string        `json:"type"`
	Payload AttachPayload `json:"payload"`
}
type AttachPayload  struct{ URL string `json:"url"` }
type ZaloUserProfile struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Picture struct{ Data struct{ URL string `json:"url"` } `json:"data"` } `json:"picture"`
	Error   int    `json:"error"`
}
type ZaloFriend struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Picture struct{ Data struct{ URL string `json:"url"` } `json:"data"` } `json:"picture"`
}
type OAuthTokenResponse struct {
	AccessToken      string `json:"access_token"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	Error            int    `json:"error"`
	ErrorDescription string `json:"error_description"`
}
