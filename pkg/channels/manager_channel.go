package channels

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
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
		hiddenValues(key, value, ch)
		valueBytes, _ := json.Marshal(value)
		hash := md5.Sum(valueBytes)
		result[key] = hex.EncodeToString(hash[:])
	}

	return result
}

func hiddenValues(key string, value map[string]any, ch config.ChannelsConfig) {
	switch key {
	case "pico":
		value["token"] = ch.Pico.Token.String()
	case "telegram":
		value["token"] = ch.Telegram.Token.String()
	case "discord":
		value["token"] = ch.Discord.Token.String()
	case "slack":
		value["bot_token"] = ch.Slack.BotToken.String()
		value["app_token"] = ch.Slack.AppToken.String()
	case "matrix":
		value["token"] = ch.Matrix.AccessToken.String()
	case "onebot":
		value["token"] = ch.OneBot.AccessToken.String()
	case "line":
		value["token"] = ch.LINE.ChannelAccessToken.String()
		value["secret"] = ch.LINE.ChannelSecret.String()
	case "wecom":
		value["secret"] = ch.WeCom.Secret.String()
	case "dingtalk":
		value["secret"] = ch.DingTalk.ClientSecret.String()
	case "qq":
		value["secret"] = ch.QQ.AppSecret.String()
	case "irc":
		value["password"] = ch.IRC.Password.String()
		value["serv_password"] = ch.IRC.NickServPassword.String()
		value["sasl_password"] = ch.IRC.SASLPassword.String()
	case "feishu":
		value["app_secret"] = ch.Feishu.AppSecret.String()
		value["encrypt_key"] = ch.Feishu.EncryptKey.String()
		value["verification_token"] = ch.Feishu.VerificationToken.String()
	case "email":
		value["smtp_password"] = ch.Email.SMTPPassword.String()
		value["imap_password"] = ch.Email.IMAPPassword.String()
	case "teams_webhook":
		// Expose webhook URLs for hash computation (they contain secrets)
		webhooks := make(map[string]string)
		for name, target := range ch.TeamsWebhook.Webhooks {
			webhooks[name] = target.WebhookURL.String()
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
	result := &config.ChannelsConfig{}
	ch := cfg.Channels
	// should not be error
	marshal, _ := json.Marshal(ch)
	var channelConfig map[string]map[string]any
	_ = json.Unmarshal(marshal, &channelConfig)
	temp := make(map[string]map[string]any, 0)

	for key, value := range channelConfig {
		found := false
		for _, s := range list {
			if key == s {
				found = true
				break
			}
		}
		if !found || !value["enabled"].(bool) {
			continue
		}
		temp[key] = value
	}

	marshal, err := json.Marshal(temp)
	if err != nil {
		logger.Errorf("marshal error: %v", err)
		return nil, err
	}
	err = json.Unmarshal(marshal, result)
	if err != nil {
		logger.Errorf("unmarshal error: %v", err)
		return nil, err
	}

	updateKeys(result, &ch)

	return result, nil
}

func updateKeys(newcfg, old *config.ChannelsConfig) {
	if newcfg.Pico.Enabled {
		newcfg.Pico.Token = old.Pico.Token
	}
	if newcfg.Telegram.Enabled {
		newcfg.Telegram.Token = old.Telegram.Token
	}
	if newcfg.Discord.Enabled {
		newcfg.Discord.Token = old.Discord.Token
	}
	if newcfg.Slack.Enabled {
		newcfg.Slack.BotToken = old.Slack.BotToken
		newcfg.Slack.AppToken = old.Slack.AppToken
	}
	if newcfg.Matrix.Enabled {
		newcfg.Matrix.AccessToken = old.Matrix.AccessToken
	}
	if newcfg.OneBot.Enabled {
		newcfg.OneBot.AccessToken = old.OneBot.AccessToken
	}
	if newcfg.LINE.Enabled {
		newcfg.LINE.ChannelAccessToken = old.LINE.ChannelAccessToken
		newcfg.LINE.ChannelSecret = old.LINE.ChannelSecret
	}
	if newcfg.WeCom.Enabled {
		newcfg.WeCom.Secret = old.WeCom.Secret
	}
	if newcfg.DingTalk.Enabled {
		newcfg.DingTalk.ClientSecret = old.DingTalk.ClientSecret
	}
	if newcfg.QQ.Enabled {
		newcfg.QQ.AppSecret = old.QQ.AppSecret
	}
	if newcfg.IRC.Enabled {
		newcfg.IRC.Password = old.IRC.Password
		newcfg.IRC.NickServPassword = old.IRC.NickServPassword
		newcfg.IRC.SASLPassword = old.IRC.SASLPassword
	}
	if newcfg.Feishu.Enabled {
		newcfg.Feishu.AppSecret = old.Feishu.AppSecret
		newcfg.Feishu.EncryptKey = old.Feishu.EncryptKey
		newcfg.Feishu.VerificationToken = old.Feishu.VerificationToken
	}
	if newcfg.Email.Enabled {
		newcfg.Email.SMTPPassword = old.Email.SMTPPassword
		newcfg.Email.IMAPPassword = old.Email.IMAPPassword
	}
	if newcfg.TeamsWebhook.Enabled {
		// Copy SecureString webhook URLs from old config
		for name, oldTarget := range old.TeamsWebhook.Webhooks {
			if newTarget, ok := newcfg.TeamsWebhook.Webhooks[name]; ok {
				newTarget.WebhookURL = oldTarget.WebhookURL
				newcfg.TeamsWebhook.Webhooks[name] = newTarget
			}
		}
	}
}
