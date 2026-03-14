package dingtalk

type ClientOption func(*Client)

func WithCardTemplateID(cardTemplateID string) ClientOption {
	return func(client *Client) {
		client.cardTemplateID = cardTemplateID
	}
}

func WithCardTemplateContentKey(cardTemplateContentKey string) ClientOption {
	return func(client *Client) {
		client.cardTemplateContentKey = cardTemplateContentKey
	}
}

func WithRobotCode(robotCode string) ClientOption {
	return func(client *Client) {
		client.robotCode = robotCode
	}
}
