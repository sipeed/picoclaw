package channels

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"

	"github.com/sipeed/picoclaw/pkg/config"
)

func toChannelHashes(cfg *config.Config) map[string]string {
	result := make(map[string]string)
	for name, bc := range cfg.Channels {
		if !bc.Enabled {
			continue
		}
		data, err := json.Marshal(bc)
		if err != nil {
			continue
		}
		hash := md5.Sum(data)
		result[name] = hex.EncodeToString(hash[:])
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
