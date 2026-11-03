package llm

var (
	defaultURL          = "https://opencode.ai/zen"
	defaultModel        = "mimo-v2.5-free"
	defaultEmbedModel   = "nomic-embed-text"
	defaultAPIKey       = ""
)

func SetDefaultConfig(url, model string) {
	if url != "" {
		defaultURL = url
	}
	if model != "" {
		defaultModel = model
	}
}

func SetDefaultAPIKey(key string) {
	if key != "" {
		defaultAPIKey = key
	}
}

func SetDefaultEmbedModel(model string) {
	if model != "" {
		defaultEmbedModel = model
	}
}

func DefaultURL() string {
	return defaultURL
}

func DefaultModel() string {
	return defaultModel
}

func DefaultAPIKey() string {
	return defaultAPIKey
}

func DefaultEmbedModel() string {
	return defaultEmbedModel
}
