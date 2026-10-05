package channels

import (
	"encoding/json"
	"testing"

	"github.com/sipeed/picoclaw/pkg/config"
)

func TestSendblueNamedInstanceSecretReloadHash(t *testing.T) {
	cfg := config.DefaultConfig()
	if err := json.Unmarshal(
		[]byte(
			`{"phone2":{"enabled":true,"type":"sendblue","allow_from":["+15550000002"],"settings":{"api_key":"key","api_secret":"secret","webhook_secret":"webhook","from_number":"+15550000001"}}}`,
		),
		&cfg.Channels,
	); err != nil {
		t.Fatal(err)
	}
	if err := config.InitChannelList(cfg.Channels); err != nil {
		t.Fatal(err)
	}
	before := toChannelHashes(cfg)["phone2"]
	raw, err := cfg.Channels["phone2"].GetDecoded()
	if err != nil {
		t.Fatal(err)
	}
	settings := raw.(*config.SendblueSettings)
	settings.APISecret.Set("rotated-secret")
	after := toChannelHashes(cfg)["phone2"]
	if before == "" || before == after {
		t.Fatal("secret rotation did not change named instance hash")
	}
}
