package services

import (
	"spiritFruit/pkg/config"
	"spiritFruit/pkg/openai"
	"strings"
)

func NewTextProvider(adminID uint64) (openai.Provider, string) {
	cfg := openai.Config{Provider: config.GetString("ai.provider", "openai"), OpenAIBaseURL: config.GetString("ai.openai.base_url"), OpenAIKey: config.GetString("ai.openai.api_key"), OpenAIModel: config.GetString("ai.openai.model"), GetGoAPIBaseURL: config.GetString("ai.getgoapi.base_url"), GetGoAPIKey: config.GetString("ai.getgoapi.api_key"), GetGoAPIModel: config.GetString("ai.getgoapi.model"), GeminiBaseURL: config.GetString("ai.gemini.base_url"), GeminiKey: config.GetString("ai.gemini.api_key"), GeminiModel: config.GetString("ai.gemini.model"), DoubaoBaseURL: config.GetString("ai.doubao.base_url"), DoubaoKey: config.GetString("ai.doubao.api_key"), DoubaoModel: config.GetString("ai.doubao.model"), VertexKey: config.GetString("ai.vertex.api_key"), VertexModel: config.GetString("ai.vertex.model"), AgnesBaseURL: config.GetString("ai.agnes.base_url"), AgnesKey: config.GetString("ai.agnes.api_key"), AgnesModel: config.GetString("ai.agnes-llm.model", "agnes-3.0-flash"), AgnesImageModel: config.GetString("ai.agnes-llm.image_model", "agnes-image-2.5-flash")}
	if err, db := new(AiConfigService).GetActiveConfigByType("text", &adminID); err == nil && db.ID > 0 {
		p := strings.ToLower(*db.Provider)
		model := ""
		if len(db.Model) > 0 {
			model = db.Model[0]
		}
		cfg.Provider = p
		switch p {
		case "getgoapi":
			cfg.GetGoAPIBaseURL = *db.BaseUrl
			cfg.GetGoAPIKey = *db.ApiKey
			cfg.GetGoAPIModel = model
		case "gemini", "google":
			cfg.Provider = "gemini"
			cfg.GeminiBaseURL = *db.BaseUrl
			cfg.GeminiKey = *db.ApiKey
			cfg.GeminiModel = model
		case "doubao", "volcengine", "volces":
			cfg.Provider = "doubao"
			cfg.DoubaoBaseURL = *db.BaseUrl
			cfg.DoubaoKey = *db.ApiKey
			cfg.DoubaoModel = model
		case "vertex":
			cfg.VertexKey = *db.ApiKey
			cfg.VertexModel = model
		case "agnes", "agnes-llm", "agnes-image":
			cfg.Provider = "agnes"
			cfg.AgnesBaseURL = *db.BaseUrl
			cfg.AgnesKey = *db.ApiKey
			cfg.AgnesModel = model
		default:
			cfg.Provider = "openai"
			cfg.OpenAIBaseURL = *db.BaseUrl
			cfg.OpenAIKey = *db.ApiKey
			cfg.OpenAIModel = model
		}
		return openai.NewProvider(cfg), p + ":" + model
	}
	return openai.NewProvider(cfg), cfg.Provider
}
