package mcp

import "encoding/json"

type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type InitializeParams struct {
	ProtocolVersion string `json:"protocolVersion"`
	ClientInfo      struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"clientInfo"`
}

type InitializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	ServerInfo      ServerInfo `json:"serverInfo"`
	Capabilities    Caps       `json:"capabilities"`
}

type ServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type Caps struct {
	Tools *ToolsCap `json:"tools,omitempty"`
}

type ToolsCap struct {
	ListChanged bool `json:"listChanged"`
}

type ToolsListResult struct {
	Tools []ToolDefinition `json:"tools"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type CallToolResult struct {
	Content           []ContentItem  `json:"content"`
	StructuredContent interface{}    `json:"structuredContent,omitempty"`
	Meta              map[string]any `json:"_meta,omitempty"`
	IsError           bool           `json:"isError,omitempty"`
}

type ContentItem struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type ToolDefinition struct {
	Name            string           `json:"name"`
	Title           string           `json:"title,omitempty"`
	Description     string           `json:"description"`
	InputSchema     json.RawMessage  `json:"inputSchema"`
	SecuritySchemes []SecurityScheme `json:"securitySchemes,omitempty"`
	Annotations     *ToolAnnotations `json:"annotations,omitempty"`
	Meta            map[string]any   `json:"_meta,omitempty"`
}

type SecurityScheme struct {
	Type   string   `json:"type"`
	Scopes []string `json:"scopes,omitempty"`
}

type ToolAnnotations struct {
	ReadOnlyHint bool `json:"readOnlyHint,omitempty"`
}

type ToolResultData struct {
	StructuredContent interface{}
	Text              string
	Meta              map[string]any
	IsError           bool
}
