package openai

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// AgnesClient Agnes 生文/生图客户端
// 文本: agnes-3.0-flash (OpenAI 兼容 chat/completions)
// 生图: agnes-image-2.5-flash (OpenAI 兼容 images/generations)
// 文档: https://www.agnes-ai.cn/zh-Hans/docs/agnes-30-flash
//       https://www.agnes-ai.cn/zh-Hans/docs/agnes-image-25-flash
type AgnesClient struct {
	Config Config
	client *http.Client
}

// agnesChatResp Agnes 文本响应 (OpenAI 兼容)
type agnesChatResp struct {
	Choices []struct {
		Message ChatMessage `json:"message"`
	} `json:"choices"`
}

// GenerateScript 实现文本/剧本生成 (OpenAI 兼容)
func (c *AgnesClient) GenerateScript(req ScriptRequest) (string, error) {
	model := c.Config.AgnesModel
	if model == "" {
		model = "agnes-3.0-flash"
	}

	payload := map[string]interface{}{
		"model":      model,
		"messages":   req.Messages,
		"max_tokens": req.MaxTokens,
	}

	headers := map[string]string{
		"Authorization": "Bearer " + c.Config.AgnesKey,
	}

	baseURL := c.Config.AgnesBaseURL
	if baseURL == "" {
		baseURL = "https://api.agnes-ai.cn/v1"
	}
	url := strings.TrimRight(baseURL, "/") + "/chat/completions"

	resp, err := doRequest[*agnesChatResp](c.client, "POST", url, headers, payload)
	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return resp.Choices[0].Message.Content, nil
	}
	return "", fmt.Errorf("agnes returned no choices")
}

// agnesImageReq Agnes 生图请求体
type agnesImageReq struct {
	Model     string `json:"model"`
	Prompt    string `json:"prompt"`
	Size      string `json:"size"`
	Ratio     string `json:"ratio,omitempty"`
	ExtraBody struct {
		ResponseFormat string `json:"response_format"`
	} `json:"extra_body"`
}

// agnesImageResp Agnes 生图响应
type agnesImageResp struct {
	Data []struct {
		URL     string `json:"url"`
		B64JSON string `json:"b64_json"`
	} `json:"data"`
}

// GenerateImage 实现生图 (Agnes 兼容 OpenAI images/generations)
func (c *AgnesClient) GenerateImage(req ImageRequest) ([]string, error) {
	model := c.Config.AgnesImageModel
	if model == "" {
		model = "agnes-image-2.5-flash"
	}

	size := req.Size
	if size == "" {
		size = "1024x1024"
	}

	payload := agnesImageReq{
		Model:  model,
		Prompt: req.Prompt,
		Size:   size,
		Ratio:  deriveAgnesRatio(size),
	}
	payload.ExtraBody.ResponseFormat = "url"

	headers := map[string]string{
		"Authorization": "Bearer " + c.Config.AgnesKey,
	}

	baseURL := c.Config.AgnesBaseURL
	if baseURL == "" {
		baseURL = "https://api.agnes-ai.cn/v1"
	}
	url := strings.TrimRight(baseURL, "/") + "/images/generations"

	resp, err := doRequest[*agnesImageResp](c.client, "POST", url, headers, payload)
	if err != nil {
		return nil, err
	}

	var urls []string
	for _, item := range resp.Data {
		if item.URL != "" {
			urls = append(urls, item.URL)
		} else if item.B64JSON != "" {
			urls = append(urls, "data:image/png;base64,"+item.B64JSON)
		}
	}

	if len(urls) == 0 {
		return nil, fmt.Errorf("agnes returned empty image list")
	}
	return urls, nil
}

// deriveAgnesRatio 从 "WxH" 尺寸推导 Agnes 支持的宽高比，无法匹配则返回空串
func deriveAgnesRatio(size string) string {
	parts := strings.Split(size, "x")
	if len(parts) != 2 {
		return ""
	}
	w, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
	h, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
	if err1 != nil || err2 != nil || w == 0 || h == 0 {
		return ""
	}
	allowed := []struct {
		r  string
		aw int
		ah int
	}{
		{"1:1", 1, 1}, {"3:4", 3, 4}, {"4:3", 4, 3}, {"16:9", 16, 9},
		{"9:16", 9, 16}, {"2:3", 2, 3}, {"3:2", 3, 2}, {"21:9", 21, 9},
	}
	for _, a := range allowed {
		if w*a.ah == h*a.aw {
			return a.r
		}
	}
	return ""
}
