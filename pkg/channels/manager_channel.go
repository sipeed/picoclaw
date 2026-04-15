package channels

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	"github.com/sipeed/picoclaw/pkg/config"
)

func toChannelHashes(cfg *config.Config) map[string]string {
	result := make(map[string]string)
	ch := cfg.Channels
	// should not be error
	marshal, _ := json.Marshal(ch)
	var channelConfig map[string]map[string]any
	_ = json.Unmarshal(marshal, &channelConfig)

	for key, value := range channelConfig {
		if !value["enabled"].(bool) {
			continue
		}
		hiddenValues(key, value, ch.Get(key))
		valueBytes, _ := json.Marshal(value)
		hash := md5.Sum(valueBytes)
		result[key] = hex.EncodeToString(hash[:])
	}

	return result
}

func hiddenValues(key string, value map[string]any, ch *config.Channel) {
	v, err := ch.GetDecoded()
	if err != nil {
		return
	}
	switch key {
	case "pico":
		if settings, ok := v.(*config.PicoSettings); ok {
			value["token"] = settings.Token.String()
		}
	case "telegram":
		if settings, ok := v.(*config.TelegramSettings); ok {
			value["token"] = settings.Token.String()
		}
	case "discord":
		if settings, ok := v.(*config.DiscordSettings); ok {
			value["token"] = settings.Token.String()
		}
	case "slack":
		if settings, ok := v.(*config.SlackSettings); ok {
			value["bot_token"] = settings.BotToken.String()
			value["app_token"] = settings.AppToken.String()
		}
	case "matrix":
		if settings, ok := v.(*config.MatrixSettings); ok {
			value["token"] = settings.AccessToken.String()
		}
	case "onebot":
		if settings, ok := v.(*config.OneBotSettings); ok {
			value["token"] = settings.AccessToken.String()
		}
	case "line":
		if settings, ok := v.(*config.LINESettings); ok {
			value["token"] = settings.ChannelAccessToken.String()
			value["secret"] = settings.ChannelSecret.String()
		}
	case "wecom":
		if settings, ok := v.(*config.WeComSettings); ok {
			value["secret"] = settings.Secret.String()
		}
	case "dingtalk":
		if settings, ok := v.(*config.DingTalkSettings); ok {
			value["secret"] = settings.ClientSecret.String()
		}
	case "qq":
		if settings, ok := v.(*config.QQSettings); ok {
			value["secret"] = settings.AppSecret.String()
		}
	case "irc":
		if settings, ok := v.(*config.IRCSettings); ok {
			value["password"] = settings.Password.String()
			value["serv_password"] = settings.NickServPassword.String()
			value["sasl_password"] = settings.SASLPassword.String()
		}
	case "feishu":
		if settings, ok := v.(*config.FeishuSettings); ok {
			value["app_secret"] = settings.AppSecret.String()
			value["encrypt_key"] = settings.EncryptKey.String()
			value["verification_token"] = settings.VerificationToken.String()
		}
	case "teams_webhook":
		// Expose webhook URLs for hash computation (they contain secrets)
		vv := value["webhooks"]
		webhooks := make(map[string]string)
		if vv != nil {
			webhooks = vv.(map[string]string)
		}
		if settings, ok := v.(*config.TeamsWebhookSettings); ok {
			for name, target := range settings.Webhooks {
				webhooks[name] = target.WebhookURL.String()
			}
		}
		value["webhooks"] = webhooks
	}
}

func compareChannels(old, news map[string]string) (added, removed []string) {
	for key, newHash := range news {
		if oldHash, ok := old[key]; ok {
			if newHash != oldHash {
				removed = append(removed, key)
				added = append(added, key)
			}
		} else {
			added = append(added, key)
		}
	}
	for key := range old {
		if _, ok := news[key]; !ok {
			removed = append(removed, key)
		}
	}
	return added, removed
}

func toChannelConfig(cfg *config.Config, list []string) (*config.ChannelsConfig, error) {
	result := make(config.ChannelsConfig)
	for _, name := range list {
		bc, ok := cfg.Channels[name]
		if !ok || !bc.Enabled {
			continue
		}
		result[name] = bc
	}
	return &result, nil
}
