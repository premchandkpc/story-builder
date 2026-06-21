package llm

var (
	defaultURL          = "http://localhost:11434"
	defaultModel        = "llama3.2:3b"
	defaultEmbedModel   = "nomic-embed-text"
)

func SetDefaultConfig(url, model string) {
	if url != "" {
		defaultURL = url
	}
	if model != "" {
		defaultModel = model
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

func DefaultEmbedModel() string {
	return defaultEmbedModel
}
