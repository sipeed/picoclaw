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
		value["token"] = ch.Pico.Token()
	case "telegram":
		value["token"] = ch.Telegram.Token()
	case "discord":
		value["token"] = ch.Discord.Token()
	case "slack":
		value["bot_token"] = ch.Slack.BotToken()
		value["app_token"] = ch.Slack.AppToken()
	case "matrix":
		value["token"] = ch.Matrix.AccessToken()
	case "onebot":
		value["token"] = ch.OneBot.AccessToken()
	case "line":
		value["token"] = ch.LINE.ChannelAccessToken()
		value["secret"] = ch.LINE.ChannelSecret()
	case "wecom":
		value["token"] = ch.WeCom.Token()
		value["key"] = ch.WeCom.EncodingAESKey()
	case "wecom_app":
		value["token"] = ch.WeComApp.Token()
		value["secret"] = ch.WeComApp.CorpSecret()
	case "wecom_aibot":
		value["token"] = ch.WeComAIBot.Token()
		value["key"] = ch.WeComAIBot.EncodingAESKey()
		value["secret"] = ch.WeComAIBot.Secret()
	case "dingtalk":
		value["secret"] = ch.QQ.AppSecret()
	case "qq":
		value["secret"] = ch.DingTalk.ClientSecret()
	case "irc":
		value["password"] = ch.IRC.Password()
		value["serv_password"] = ch.IRC.NickServPassword()
		value["sasl_password"] = ch.IRC.SASLPassword()
	case "feishu":
		value["app_secret"] = ch.Feishu.AppSecret()
		value["encrypt_key"] = ch.Feishu.EncryptKey()
		value["verification_token"] = ch.Feishu.VerificationToken()
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
		newcfg.Pico.SetToken(old.Pico.Token())
	}
	if newcfg.Telegram.Enabled {
		newcfg.Telegram.SetToken(old.Telegram.Token())
	}
	if newcfg.Discord.Enabled {
		newcfg.Discord.SetToken(old.Discord.Token())
	}
	if newcfg.Slack.Enabled {
		newcfg.Slack.SetBotToken(old.Slack.BotToken())
		newcfg.Slack.SetAppToken(old.Slack.AppToken())
	}
	if newcfg.Matrix.Enabled {
		newcfg.Matrix.SetAccessToken(old.Matrix.AccessToken())
	}
	if newcfg.OneBot.Enabled {
		newcfg.OneBot.SetAccessToken(old.OneBot.AccessToken())
	}
	if newcfg.LINE.Enabled {
		newcfg.LINE.SetChannelAccessToken(old.LINE.ChannelAccessToken())
		newcfg.LINE.SetChannelSecret(old.LINE.ChannelSecret())
	}
	if newcfg.WeCom.Enabled {
		newcfg.WeCom.SetToken(old.WeCom.Token())
		newcfg.WeCom.SetEncodingAESKey(old.WeCom.EncodingAESKey())
	}
	if newcfg.WeComApp.Enabled {
		newcfg.WeComApp.SetToken(old.WeComApp.Token())
		newcfg.WeComApp.SetCorpSecret(old.WeComApp.CorpSecret())
	}
	if newcfg.WeComAIBot.Enabled {
		newcfg.WeComAIBot.SetToken(old.WeComAIBot.Token())
		newcfg.WeComAIBot.SetEncodingAESKey(old.WeComAIBot.EncodingAESKey())
	}
	if newcfg.DingTalk.Enabled {
		newcfg.DingTalk.SetClientSecret(old.DingTalk.ClientSecret())
	}
	if newcfg.QQ.Enabled {
		newcfg.QQ.SetAppSecret(old.QQ.AppSecret())
	}
	if newcfg.IRC.Enabled {
		newcfg.IRC.SetPassword(old.IRC.Password())
		newcfg.IRC.SetNickServPassword(old.IRC.NickServPassword())
		newcfg.IRC.SetSASLPassword(old.IRC.SASLPassword())
	}
	if newcfg.Feishu.Enabled {
		newcfg.Feishu.SetAppSecret(old.Feishu.AppSecret())
		newcfg.Feishu.SetEncryptKey(old.Feishu.EncryptKey())
		newcfg.Feishu.SetVerificationToken(old.Feishu.VerificationToken())
	}
}
