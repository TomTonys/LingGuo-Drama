package video

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// AgnesClient Agnes 视频生成客户端 (agnes-video-2.5-flash)
// 文档: https://www.agnes-ai.cn/zh-Hans/docs/agnes-video-25-flash
type AgnesClient struct {
	BaseURL       string // 创建任务基址，如 https://api.agnes-ai.cn/v1
	APIKey        string
	Model         string
	Endpoint      string // 创建任务路径，如 /videos
	QueryEndpoint string // 查询任务基址，如 https://api.agnes-ai.cn/agnesapi
	HTTPClient    *http.Client
}

func NewAgnesClient(baseURL, apiKey, model, endpoint, queryEndpoint string) *AgnesClient {
	if baseURL == "" {
		baseURL = "https://api.agnes-ai.cn/v1"
	}
	if endpoint == "" {
		endpoint = "/videos"
	}
	if queryEndpoint == "" {
		queryEndpoint = "https://api.agnes-ai.cn/agnesapi"
	}
	if model == "" {
		model = "agnes-video-2.5-flash"
	}
	return &AgnesClient{
		BaseURL:       strings.TrimRight(baseURL, "/"),
		APIKey:        apiKey,
		Model:         model,
		Endpoint:      endpoint,
		QueryEndpoint: strings.TrimRight(queryEndpoint, "/"),
		HTTPClient:    defaultHTTPClient(),
	}
}

// agnesCreateRequest Agnes 创建任务请求体
type agnesCreateRequest struct {
	Model       string   `json:"model"`
	Prompt      string   `json:"prompt"`
	Mode        string   `json:"mode"` // text | keyframe | reference
	Seconds     string   `json:"seconds,omitempty"`
	Size        string   `json:"size"`
	AspectRatio string   `json:"aspect_ratio,omitempty"`
	Seed        int64    `json:"seed,omitempty"`
	N           int      `json:"n,omitempty"`
	FirstFrame  string   `json:"first_frame,omitempty"`
	LastFrame   string   `json:"last_frame,omitempty"`
	Images      []string `json:"images,omitempty"`
	Audios      []string `json:"audios,omitempty"`
}

// agnesCreateResponse Agnes 创建任务响应
type agnesCreateResponse struct {
	ID      string `json:"id"`
	TaskID  string `json:"task_id"`
	VideoID string `json:"video_id"`
	Status  string `json:"status"`
	Error   string `json:"error"`
}

// GenerateVideo 发起生成请求，统一使用 VideoOption
func (c *AgnesClient) GenerateVideo(prompt string, opts ...VideoOption) (*VideoResult, error) {
	options := &VideoOptions{
		Duration:    5,
		AspectRatio: "16:9",
	}
	for _, opt := range opts {
		opt(options)
	}

	model := c.Model
	if options.Model != "" {
		model = options.Model
	}

	req := agnesCreateRequest{
		Model:       model,
		Prompt:      prompt,
		Mode:        "text",
		Seconds:     fmt.Sprintf("%d", options.Duration),
		Size:        "720P", // Flash 模型固定 720P
		AspectRatio: options.AspectRatio,
		N:           1,
	}

	// 推断模式：首尾帧 > 参考图 > 纯文本
	if options.FirstFrameURL != "" || options.LastFrameURL != "" {
		req.Mode = "keyframe"
		req.FirstFrame = options.FirstFrameURL
		req.LastFrame = options.LastFrameURL
	} else if len(options.ReferenceImageURLs) > 0 {
		req.Mode = "reference"
		req.Images = options.ReferenceImageURLs
		req.Prompt = injectPictureRefs(prompt, len(options.ReferenceImageURLs))
	} else if options.ImageURL != "" {
		req.Mode = "reference"
		req.Images = []string{options.ImageURL}
		req.Prompt = injectPictureRefs(prompt, 1)
	}

	if options.Seed != 0 {
		req.Seed = options.Seed
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	endpoint := c.BaseURL + c.Endpoint
	req2, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req2)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("agnes API error (status %d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("[Agnes] Create response: %s\n", string(body))

	var result agnesCreateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w, body: %s", err, string(body))
	}

	if result.Error != "" {
		return nil, fmt.Errorf("agnes error: %s", result.Error)
	}

	// 优先 video_id，其次 id / task_id
	taskID := result.VideoID
	if taskID == "" {
		taskID = result.ID
	}
	if taskID == "" {
		taskID = result.TaskID
	}
	if taskID == "" {
		return nil, fmt.Errorf("agnes response missing video_id: %s", string(body))
	}

	status := result.Status
	if status == "" {
		status = "processing"
	}

	return &VideoResult{
		TaskID:    taskID,
		Status:    status,
		Completed: status == "completed" || status == "succeeded",
		Duration:  options.Duration,
	}, nil
}

// GetTaskStatus 查询任务状态，统一使用 VideoResult
func (c *AgnesClient) GetTaskStatus(taskID string) (*VideoResult, error) {
	queryURL := fmt.Sprintf("%s?video_id=%s&model_name=%s", c.QueryEndpoint, taskID, c.Model)

	req, err := http.NewRequest("GET", queryURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.APIKey)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("agnes query error (status %d): %s", resp.StatusCode, string(body))
	}

	fmt.Printf("[Agnes] Query response: %s\n", string(body))

	// 解析为通用 map，灵活提取 status 与 video url
	var data map[string]interface{}
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("parse query response: %w, body: %s", err, string(body))
	}

	status := strings.ToLower(strings.TrimSpace(extractString(data, "status")))
	videoURL := extractVideoURL(data)

	completed := status == "completed" || status == "succeeded"
	failed := status == "failed" || status == "error" || status == "expired"

	result := &VideoResult{
		TaskID:    taskID,
		Status:    status,
		Completed: completed,
		VideoURL:  videoURL,
	}

	if failed {
		if msg := extractString(data, "error", "message", "err_msg", "reason"); msg != "" {
			result.Error = msg
		} else {
			result.Error = "agnes task failed"
		}
	}

	return result, nil
}

// injectPictureRefs 在 prompt 中注入 <Picture N> 引用（reference 模式需要）
func injectPictureRefs(prompt string, n int) string {
	if strings.Contains(prompt, "<Picture") {
		return prompt
	}
	var refs []string
	for i := 1; i <= n; i++ {
		refs = append(refs, fmt.Sprintf("<Picture %d>", i))
	}
	return fmt.Sprintf("%s\n参考素材: %s", prompt, strings.Join(refs, ", "))
}

// extractString 从通用 map 中按候选 key（大小写不敏感）提取字符串值
func extractString(data map[string]interface{}, keys ...string) string {
	for _, k := range keys {
		for dk, dv := range data {
			if strings.EqualFold(dk, k) {
				if s, ok := dv.(string); ok && s != "" {
					return s
				}
			}
		}
	}
	return ""
}

// extractVideoURL 从通用 map 中灵活提取视频 URL
func extractVideoURL(data map[string]interface{}) string {
	// 1. 已知 key 优先
	candidates := []string{
		"video_url", "url", "video", "output", "output_url",
		"result", "play_url", "download_url", "mp4_url", "src",
	}
	for _, k := range candidates {
		for dk, dv := range data {
			if strings.EqualFold(dk, k) {
				if s, ok := dv.(string); ok && isURL(s) {
					return s
				}
			}
		}
	}

	// 2. 递归搜索，优先带视频扩展名的 URL
	var videoURL, anyURL string
	var walk func(m map[string]interface{})
	walk = func(m map[string]interface{}) {
		for _, v := range m {
			switch val := v.(type) {
			case string:
				if isURL(val) {
					if anyURL == "" {
						anyURL = val
					}
					if strings.Contains(val, ".mp4") || strings.Contains(val, ".mov") ||
						strings.Contains(val, ".webm") || strings.Contains(val, "video") {
						videoURL = val
					}
				}
			case map[string]interface{}:
				walk(val)
			case []interface{}:
				for _, item := range val {
					if sm, ok := item.(map[string]interface{}); ok {
						walk(sm)
					} else if s, ok := item.(string); ok && isURL(s) {
						if anyURL == "" {
							anyURL = s
						}
						if strings.Contains(s, ".mp4") || strings.Contains(s, ".mov") ||
							strings.Contains(s, ".webm") || strings.Contains(s, "video") {
							videoURL = s
						}
					}
				}
			}
		}
	}
	walk(data)

	if videoURL != "" {
		return videoURL
	}
	return anyURL
}

func isURL(s string) bool {
	return strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://")
}
