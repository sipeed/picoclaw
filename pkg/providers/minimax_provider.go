package providers

func MiniMaxProvider(apiKey, apiBase string) *HTTPProvider {
	p := NewHTTPProvider(apiKey, apiBase, "")
	p.RequestSuffix = "/text/chatcompletion_v2"
	return p
}
