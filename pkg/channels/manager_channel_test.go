package channels

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/sipeed/picoclaw/pkg/config"
	"github.com/sipeed/picoclaw/pkg/logger"
)

func TestToChannelHashes(t *testing.T) {
	logger.SetLevel(logger.DEBUG)
	cfg := config.DefaultConfig()
	results := toChannelHashes(cfg)
	assert.Equal(t, 0, len(results))
	logger.Debugf("results: %v", results)
	cfg2 := config.DefaultConfig()
	cfg2.Channels.DingTalk.Enabled = true
	results2 := toChannelHashes(cfg2)
	assert.Equal(t, 1, len(results2))
	logger.Debugf("results2: %v", results2)
	added, removed := compareChannels(results, results2)
	assert.EqualValues(t, []string{"dingtalk"}, added)
	assert.EqualValues(t, []string(nil), removed)
	cfg3 := config.DefaultConfig()
	cfg3.Channels.Telegram.Enabled = true
	results3 := toChannelHashes(cfg3)
	assert.Equal(t, 1, len(results3))
	logger.Debugf("results3: %v", results3)
	added, removed = compareChannels(results2, results3)
	assert.EqualValues(t, []string{"dingtalk"}, removed)
	assert.EqualValues(t, []string{"telegram"}, added)
	cfg3.Channels.Telegram.SetToken("114314")
	results4 := toChannelHashes(cfg3)
	assert.Equal(t, 1, len(results4))
	logger.Debugf("results4: %v", results4)
	added, removed = compareChannels(results3, results4)
	assert.EqualValues(t, []string{"telegram"}, removed)
	assert.EqualValues(t, []string{"telegram"}, added)
	cc, err := toChannelConfig(cfg3, added)
	assert.NoError(t, err)
	logger.Debugf("cc: %#v", cc.Telegram)
	assert.Equal(t, "114314", cc.Telegram.Token())
	assert.Equal(t, true, cc.Telegram.Enabled)
	cc, err = toChannelConfig(cfg2, added)
	assert.NoError(t, err)
	logger.Debugf("cc: %#v", cc.Telegram)
	assert.Equal(t, "", cc.Telegram.Token())
	assert.Equal(t, false, cc.Telegram.Enabled)
}
