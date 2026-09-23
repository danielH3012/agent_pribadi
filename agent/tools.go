package agent



type ToolDefinition struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolFunction mendeskripsikan satu function beserta parameter-nya.
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  *FunctionParams `json:"parameters,omitempty"`
}

// FunctionParams adalah JSON Schema untuk parameter function.
type FunctionParams struct {
	Type                 string                    `json:"type"`
	Properties           map[string]PropertySchema `json:"properties,omitempty"`
	Required             []string                  `json:"required,omitempty"`
	AdditionalProperties bool                      `json:"additionalProperties"`
}

// PropertySchema mendeskripsikan satu properti di dalam "properties".
type PropertySchema struct {
	Type        string                    `json:"type"`
	Description string                    `json:"description,omitempty"`
	Items       *PropertySchema           `json:"items,omitempty"`
	Properties  map[string]PropertySchema `json:"properties,omitempty"`
	Required    []string                  `json:"required,omitempty"`
}

// Tools berisi daftar definisi tool yang kompatibel dengan format OpenAI / LLM function calling.
var Tools = []ToolDefinition{
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "bash",
			Description: "Execute a shell command and capture its stdout, exit code, and status.",
			Parameters: &FunctionParams{
				Type: "object",
				Properties: map[string]PropertySchema{
					"command": {
						Type:        "string",
						Description: "The shell command to execute.",
					},
				},
				Required:             []string{"command"},
				AdditionalProperties: false,
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "edit",
			Description: "Edit a file by replacing old content with new content atomically.",
			Parameters: &FunctionParams{
				Type: "object",
				Properties: map[string]PropertySchema{
					"path": {
						Type:        "string",
						Description: "The path of the file to edit.",
					},
					"old_content": {
						Type:        "string",
						Description: "The exact text/content in the file to be replaced.",
					},
					"new_content": {
						Type:        "string",
						Description: "The new text/content that will replace old_content.",
					},
				},
				Required:             []string{"path", "old_content", "new_content"},
				AdditionalProperties: false,
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "glob",
			Description: "Find files matching a glob pattern (supports *, **, ?, and character classes).",
			Parameters: &FunctionParams{
				Type: "object",
				Properties: map[string]PropertySchema{
					"pattern": {
						Type:        "string",
						Description: "The glob pattern to match files against (e.g. '**/*.go', 'tools/*.go').",
					},
				},
				Required:             []string{"pattern"},
				AdditionalProperties: false,
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "grep",
			Description: "Search for regular expression pattern matches within files in a directory.",
			Parameters: &FunctionParams{
				Type: "object",
				Properties: map[string]PropertySchema{
					"pattern": {
						Type:        "string",
						Description: "The regular expression pattern to search for.",
					},
					"path": {
						Type:        "string",
						Description: "The directory path to search within.",
					},
					"context_lines": {
						Type:        "integer",
						Description: "Optional number of context lines before and after each match.",
					},
				},
				Required:             []string{"pattern", "path"},
				AdditionalProperties: false,
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "multi_edit",
			Description: "Perform multiple file edit operations atomically with rollback on any failure.",
			Parameters: &FunctionParams{
				Type: "object",
				Properties: map[string]PropertySchema{
					"operations": {
						Type:        "array",
						Description: "List of edit operations to apply atomically.",
						Items: &PropertySchema{
							Type:        "object",
							Description: "Single file edit operation.",
							Properties: map[string]PropertySchema{
								"path": {
									Type:        "string",
									Description: "The path of the file to edit.",
								},
								"old_content": {
									Type:        "string",
									Description: "The exact text to be replaced in the file.",
								},
								"new_content": {
									Type:        "string",
									Description: "The new text to replace old_content.",
								},
							},
							Required: []string{"path", "old_content", "new_content"},
						},
					},
				},
				Required:             []string{"operations"},
				AdditionalProperties: false,
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "read",
			Description: "Read the contents of a file with line numbers, optionally limiting the number of lines.",
			Parameters: &FunctionParams{
				Type: "object",
				Properties: map[string]PropertySchema{
					"path": {
						Type:        "string",
						Description: "The path of the file to read.",
					},
					"limit": {
						Type:        "integer",
						Description: "Optional maximum number of lines to read.",
					},
				},
				Required:             []string{"path"},
				AdditionalProperties: false,
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "write",
			Description: "Create a new file and write content to it atomically (fails if file already exists).",
			Parameters: &FunctionParams{
				Type: "object",
				Properties: map[string]PropertySchema{
					"path": {
						Type:        "string",
						Description: "The path of the file to create and write to.",
					},
					"content": {
						Type:        "string",
						Description: "The text content to write into the file.",
					},
				},
				Required:             []string{"path", "content"},
				AdditionalProperties: false,
			},
		},
	},
}

// tools is alias for internal package compatibility.
var tools = Tools

// GetTools returns the list of all tool definitions.
func GetTools() []ToolDefinition {
	return Tools
}

// Definisi masing-masing tool agar dapat langsung dipilih/diberikan dalam bentuk array [tool1, tool2, tool3].
var (
	BashTool      = Tools[0]
	EditTool      = Tools[1]
	GlobTool      = Tools[2]
	GrepTool      = Tools[3]
	MultiEditTool = Tools[4]
	ReadTool      = Tools[5]
	WriteTool     = Tools[6]
)
