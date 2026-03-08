package dingtalk

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/open-dingtalk/dingtalk-stream-sdk-go/chatbot"
)

const (
	endpoint             = "https://api.dingtalk.com"
	accessToken          = "/v1.0/oauth2/accessToken"
	createCardAndDeliver = "/v1.0/card/instances/createAndDeliver"
	cardStreaming        = "/v1.0/card/streaming"
	privateChatMessages  = "/v1.0/robot/privateChatMessages/send"
	batchSendMessages    = "/v1.0/robot/oToMessages/batchSend"
)

// MessageType represents the type of message to be sent.
type MessageType string

func (t MessageType) String() string {
	return string(t)
}

const (
	Markdown = MessageType("sampleMarkdown")
	Text     = MessageType("sampleText")
	Image    = MessageType("sampleImageMsg")
)

type Client struct {
	clientID     string
	clientSecret string
	token        string
	// card template id
	cardTemplateID string
	// card template message content default content
	cardTemplateContentKey string

	// robot code default use clientID
	robotCode string

	expires time.Time
	client  *http.Client
}

// NewClient creates a new Client instance with the provided client ID, client secret, and optional configurations.
func NewClient(clientID, clientSecret string, opts ...ClientOption) *Client {
	client := &Client{clientID: clientID, clientSecret: clientSecret,
		client: &http.Client{Timeout: time.Second * 30}}
	for _, opt := range opts {
		opt(client)
	}
	if client.robotCode == "" {
		client.robotCode = client.clientID
	}
	if client.cardTemplateContentKey == "" {
		client.cardTemplateContentKey = "content"
	}
	return client
}

// GetToken retrieves the access token, refreshing it if it has expired.
func (c *Client) GetToken() (string, error) {
	if time.Now().Before(c.expires) {
		return c.token, nil
	}
	data := map[string]any{
		"appKey":    c.clientID,
		"appSecret": c.clientSecret,
	}
	resp := struct {
		Expires int64  `json:"expireIn"`
		Token   string `json:"accessToken"`
	}{}
	err := c.httpRequest(context.Background(), http.MethodPost, accessToken, data, &resp)
	if err != nil {
		return "", err
	}
	c.expires = time.Now().Add(time.Second * time.Duration(resp.Expires))
	c.token = resp.Token
	return c.token, nil
}

// BatchSendMessages sends a message to multiple users in a batch.
func (c *Client) BatchSendMessages(ctx context.Context, msgType MessageType, userIds []string, content string) error {
	body := map[string]any{
		"robotCode": c.robotCode,
		"msgKey":    msgType,
		"userIds":   userIds,
		"msgParam":  c.buildSendMessages(msgType, content),
	}
	return c.httpRequest(ctx, http.MethodPost, batchSendMessages, body, nil)
}

// CardStreaming updates the content of a card instance identified by cardInstanceId.
func (c *Client) CardStreaming(ctx context.Context, cardInstanceId, content string) error {
	id, err := uuid.NewUUID()
	if err != nil {
		return err
	}
	body := map[string]any{
		"outTrackId": cardInstanceId,
		"guid":       id.String(),
		"key":        c.cardTemplateContentKey,
		"content":    content,
		"isFull":     true,
	}
	return c.httpRequest(ctx, http.MethodPut, cardStreaming, body, nil)
}

// CardCreateAndDeliver creates a card instance and delivers it to the user or group.
// It returns the outTrackId of the created card instance, which can be used for subsequent updates via CardStreaming.
// If the chatbot is in a group conversation, the card will be delivered to the group; otherwise, it will be delivered to the user.
// <a href="https://open.dingtalk.com/document/development/create-and-deliver-cards">Card Delivery API Documentation</a>
func (c *Client) CardCreateAndDeliver(ctx context.Context, chatbot *chatbot.BotCallbackDataModel) (string, error) {
	if c.cardTemplateID == "" {
		return "", errors.New("cardTemplateId is required")
	}
	var (
		group                   = chatbot.ConversationType == "2"
		openSpaceId             = "dtv1.card//IM_ROBOT." + chatbot.SenderStaffId
		imRobotOpenDeliverModel = map[string]any{
			"spaceType": "IM_ROBOT",
		}
		imGroupOpenSpaceModel = map[string]any{
			"supportForward": true,
		}
		imRobotOpenSpaceModel = map[string]any{
			"supportForward": true,
		}
		imGroupOpenDeliverModel = map[string]any{
			"robotCode": c.clientID,
		}
	)

	id, err := uuid.NewUUID()
	if err != nil {
		return "", err
	}
	if group {
		openSpaceId = "dtv1.card//IM_GROUP." + chatbot.ConversationId
		imRobotOpenDeliverModel = map[string]any{}
		imGroupOpenDeliverModel["robotCode"] = c.clientID
	}

	body := map[string]any{
		"cardTemplateId":          c.cardTemplateID,
		"outTrackId":              id.String(),
		"cardData":                map[string]any{},
		"openSpaceId":             openSpaceId,
		"userIdType":              1,
		"imGroupOpenDeliverModel": imGroupOpenDeliverModel,
		"imGroupOpenSpaceModel":   imGroupOpenSpaceModel,
		"imRobotOpenSpaceModel":   imRobotOpenSpaceModel,
		"imRobotOpenDeliverModel": imRobotOpenDeliverModel,
	}
	/**
	{
	  "result" : {
	    "deliverResults" : [ {
	      "spaceId" : "manager164",
	      "spaceType" : "IM_ROBOT",
	      "success" : true,
	      "carrierId" : "119X11tauuJlODPiK0wpCjIXPcGpODOnnpHc/uYFFnI=",
	      "errorMsg" : ""
	    } ],
	    "outTrackId" : "f51222b2-1aff-11f1-8a7c-a40c662198be"
	  },
	  "success" : true
	}
	*/
	resp := struct {
		Result struct {
			DeliverResults []struct {
				SpaceId   string `json:"spaceId"`
				SpaceType string `json:"spaceType"`
				Success   bool   `json:"success"`
				CarrierId string `json:"carrierId"`
				ErrorMsg  string `json:"errorMsg"`
			} `json:"deliverResults"`
			OutTrackId string `json:"outTrackId"`
		} `json:"result"`
		Success bool `json:"success"`
	}{}
	if err = c.httpRequest(ctx, http.MethodPost, createCardAndDeliver, body, &resp); err != nil {
		return "", err
	}
	if resp.Success {
		return resp.Result.OutTrackId, nil
	}
	return "", errors.New("failed to create and deliver card instance")
}

// PrivateChatMessages sends a message to a user in a private chat.
func (c *Client) PrivateChatMessages(ctx context.Context, msgType MessageType, openConversationId, content string) error {
	body := map[string]any{
		"msgKey":             msgType,
		"msgParam":           c.buildSendMessages(msgType, content),
		"openConversationId": openConversationId,
		"robotCode":          c.robotCode,
	}
	return c.httpRequest(ctx, http.MethodPost, privateChatMessages, body, nil)
}

func (c *Client) buildSendMessages(msgType MessageType, content string) string {
	data := map[string]any{}
	switch msgType {
	case Markdown:
		data = map[string]any{
			"title": "PicoClaw",
			"text":  content,
		}
	case Text:
		data = map[string]any{
			"content": content,
		}
	}
	msg, _ := json.Marshal(data)
	return string(msg)
}

func (c *Client) httpRequest(ctx context.Context, method, path string, body interface{}, resp interface{}) error {
	var (
		err   error
		token string
		req   *http.Request
		hc    = c.client
		res   *http.Response
	)
	if path == accessToken {
		token = ""
	} else {
		token, err = c.GetToken()
		if err != nil {
			return err
		}
	}
	url := endpoint + path
	if body != nil {
		data, _ := json.Marshal(body)
		req, err = http.NewRequestWithContext(ctx, method, url, bytes.NewReader(data))
	} else {
		req, err = http.NewRequestWithContext(ctx, method, url, nil)
	}
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("x-acs-dingtalk-access-token", token)
	}
	res, err = hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK || err != nil {
		return fmt.Errorf("API request failed:\n  Status: %d\n  Body:   %s", res.StatusCode, string(data))
	}
	if resp == nil {
		return nil
	}
	return json.Unmarshal(data, resp)
}
