module github.com/sipeed/picoclaw/picoclaw-voice

go 1.25.7

require (
	github.com/caarlos0/env/v11 v11.3.1
	github.com/google/uuid v1.6.0
	github.com/gorilla/websocket v1.5.3
	github.com/pion/opus v0.0.0-20260219180131-abe26becac00
	github.com/sipeed/picoclaw v0.0.0
)

require (
	github.com/adhocore/gronx v1.19.6 // indirect
	github.com/anthropics/anthropic-sdk-go v1.22.1 // indirect
	github.com/github/copilot-sdk/go v0.1.23 // indirect
	github.com/gomarkdown/markdown v0.0.0-20260217112301-37c66b85d6ab // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/h2non/filetype v1.1.3 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/modelcontextprotocol/go-sdk v1.3.1 // indirect
	github.com/openai/openai-go/v3 v3.22.0 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/segmentio/asm v1.1.3 // indirect
	github.com/segmentio/encoding v0.5.3 // indirect
	github.com/tidwall/gjson v1.18.0 // indirect
	github.com/tidwall/match v1.2.0 // indirect
	github.com/tidwall/pretty v1.2.1 // indirect
	github.com/tidwall/sjson v1.2.5 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	golang.org/x/oauth2 v0.35.0 // indirect
	golang.org/x/sys v0.41.0 // indirect
	golang.org/x/time v0.14.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

// 指向本地 picoclaw fork 根目录
replace github.com/sipeed/picoclaw => ../../
