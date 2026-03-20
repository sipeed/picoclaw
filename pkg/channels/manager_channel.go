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
		valueBytes, _ := json.Marshal(value)
		hash := md5.Sum(valueBytes)
		result[key] = hex.EncodeToString(hash[:])
	}

	return result
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

	return result, nil
}
