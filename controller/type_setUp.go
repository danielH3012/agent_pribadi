package agent

import (
	"encoding/json"
	"strings"
)

type UserAuth struct {
	Role      string `json:"role"`
	UserID    string `json:"user_id"`
	CompanyID string `json:"company_id"`
	Name      string `json:"name"`
}

type RequestChat struct {
	Auth       UserAuth `json:"auth"` // Disimpan agar RBAC & Company scoping bisa dibaca engine
	CacheID    string   `json:"cache_id"`
	Chat       string   `json:"chat"`                  // Pesan / query dari user
	Model      string   `json:"model,omitempty"`       // Model override (opsional)
	Attach     bool     `json:"attach"`                // Status ada/tidaknya lampiran
	LinkAttach string   `json:"link_attach,omitempty"` // Path lokal / URL file lampiran
	API_KEY    string   `json:"api_key"`
	LLM_URl    string   `json:"llm_url"`
}

type RequestChatOption func(*RequestChat)

// Opsi untuk custom CacheID / Session ID
func WithCacheID(id string) RequestChatOption {
	return func(r *RequestChat) {
		if id != "" {
			r.CacheID = id
		}
	}
}

// Opsi untuk menyertakan lampiran file
func WithAttachment(filePath string) RequestChatOption {
	return func(r *RequestChat) {
		if filePath != "" && filePath != "-" {
			r.Attach = true
			r.LinkAttach = filePath
		}
	}
}

// Opsi untuk override model LLM
func WithModel(model string, apiKey string, llmUrl string) RequestChatOption {
	return func(r *RequestChat) {
		if model != "" {
			r.Model = model
		}
		if apiKey != "" {
			r.API_KEY = apiKey
		}
		if llmUrl != "" {
			r.LLM_URl = llmUrl
		}
	}
}

type AttachmentInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
	Type string `json:"type"`
	Size int64  `json:"size"`
}

type ToolExecutionResult struct {
	Tool   string         `json:"tool"`
	Args   map[string]any `json:"args"`
	Result any            `json:"result"`
}

// AgentResult represents the output of CallTools.
type AgentResult struct {
	Context     string                `json:"context"`
	Attachment  *AttachmentInfo       `json:"attachment,omitempty"`
	ToolResults []ToolExecutionResult `json:"tool_results,omitempty"`
}

func (r *AgentResult) ToJSON() (string, error) {
	if r == nil {
		return "{}", nil
	}
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *AgentResult) ToJSONIndent() (string, error) {
	if r == nil {
		return "{}", nil
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *AgentResult) String() string {
	s, _ := r.ToJSON()
	return s
}

// ResponseResult represents the output of GenerateResponse.
type ResponseResult struct {
	Answer string `json:"answer"`
}

func (r *ResponseResult) ToJSON() (string, error) {
	if r == nil {
		return "{}", nil
	}
	b, err := json.Marshal(r)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *ResponseResult) ToJSONIndent() (string, error) {
	if r == nil {
		return "{}", nil
	}
	b, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (r *ResponseResult) String() string {
	s, _ := r.ToJSON()
	return s
}

type ChatMessage struct {
	Role string `json:"role"`
	Chat string `json:"chat"`
}

// FormatChatHistoryForLlm formats chat history messages into a dialog block for the LLM
func FormatChatHistoryForLlm(history []ChatMessage) string {
	if len(history) == 0 {
		return ""
	}
	var lines []string
	for _, msg := range history {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "User"
		}
		lines = append(lines, role+": "+msg.Chat)
	}
	return strings.Join(lines, "\n")
}

type ResponsePrompt struct {
	Prompt string `json:"prompt"`
}

type ToolsPrompt struct {
	Prompt string   `json:"prompt"`
	Tools  []string `json:"tools,omitempty"`
}

type RequestResponseOption func(*ResponsePrompt)
type RequestToolsOption func(*ToolsPrompt)

// WithResponsePrompt sets custom response generation instructions
func WithResponsePrompt(prompt string) RequestResponseOption {
	return func(r *ResponsePrompt) {
		if prompt != "" {
			r.Prompt = prompt
		}
	}
}

// NewResponsePrompt constructs a ResponsePrompt with options
func NewResponsePrompt(opts ...RequestResponseOption) ResponsePrompt {
	r := ResponsePrompt{}
	for _, opt := range opts {
		opt(&r)
	}
	return r
}

// WithToolsPrompt sets custom tool calling prompt / guardrails
func WithToolsPrompt(prompt string) RequestToolsOption {
	return func(r *ToolsPrompt) {
		if prompt != "" {
			r.Prompt = prompt
		}
	}
}

// WithToolsList sets specific allowed tools list
func WithToolsList(tools []string) RequestToolsOption {
	return func(r *ToolsPrompt) {
		if len(tools) > 0 {
			r.Tools = tools
		}
	}
}

// NewToolsPrompt constructs a ToolsPrompt with options
func NewToolsPrompt(opts ...RequestToolsOption) ToolsPrompt {
	tp := ToolsPrompt{}
	for _, opt := range opts {
		opt(&tp)
	}
	return tp
}

type MCPLink struct {
	Link string `json:"link"`
}

type RequestMCPOption func(*MCPLink)

// WithMCPLink sets the server URL / endpoint in MCPLink
func WithMCPLink(link string) RequestMCPOption {
	return func(m *MCPLink) {
		if link != "" {
			m.Link = link
		}
	}
}

// NewMCPLink creates an MCPLink with user options
func NewMCPLink(opts ...RequestMCPOption) MCPLink {
	m := MCPLink{}
	for _, opt := range opts {
		opt(&m)
	}
	return m
}

type subAgent struct {
	description string
	tools       []string
	prompt      string
}
