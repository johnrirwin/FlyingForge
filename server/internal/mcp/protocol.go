package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/johnrirwin/flyingforge/internal/logging"
)

const protocolVersion = "2025-06-18"

type Protocol struct {
	handler *Handler
	logger  *logging.Logger
}

func NewProtocol(handler *Handler, logger *logging.Logger) *Protocol {
	return &Protocol{
		handler: handler,
		logger:  logger,
	}
}

func (p *Protocol) HandleMessage(ctx context.Context, data []byte) *Response {
	var req Request
	if err := json.Unmarshal(data, &req); err != nil {
		return &Response{
			JSONRPC: "2.0",
			Error: &RPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		}
	}

	p.logger.Debug("Received MCP request", logging.WithFields(map[string]interface{}{
		"method": req.Method,
		"id":     req.ID,
	}))

	switch req.Method {
	case "initialize":
		return p.handleInitialize(req)
	case "initialized":
		return nil
	case "tools/list":
		return p.handleToolsList(req)
	case "tools/call":
		return p.handleToolsCall(ctx, req)
	case "ping":
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		}
	default:
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

func (p *Protocol) handleInitialize(req Request) *Response {
	result := InitializeResult{
		ProtocolVersion: protocolVersion,
		ServerInfo: ServerInfo{
			Name:    "flyingforge",
			Version: "1.0.0",
		},
		Capabilities: Caps{
			Tools: &ToolsCap{ListChanged: false},
		},
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

func (p *Protocol) handleToolsList(req Request) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  ToolsListResult{Tools: p.handler.GetTools()},
	}
}

func (p *Protocol) handleToolsCall(ctx context.Context, req Request) *Response {
	var params CallToolParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid params: " + err.Error(),
			},
		}
	}

	result, err := p.handler.HandleToolCall(ctx, params.Name, params.Arguments)
	if err != nil {
		return &Response{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: CallToolResult{
				Content: []ContentItem{{
					Type: "text",
					Text: fmt.Sprintf(`{"error":%q}`, err.Error()),
				}},
				Meta:    authMetaFromContext(ctx),
				IsError: true,
			},
		}
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  normalizeToolResult(ctx, result),
	}
}

func normalizeToolResult(ctx context.Context, result interface{}) CallToolResult {
	switch typed := result.(type) {
	case ToolResultData:
		return buildToolCallResult(ctx, typed)
	case *ToolResultData:
		if typed == nil {
			return CallToolResult{}
		}
		return buildToolCallResult(ctx, *typed)
	default:
		return buildToolCallResult(ctx, ToolResultData{
			StructuredContent: result,
			Text:              compactJSON(result),
		})
	}
}

func buildToolCallResult(ctx context.Context, data ToolResultData) CallToolResult {
	meta := map[string]any{}
	for key, value := range authMetaFromContext(ctx) {
		meta[key] = value
	}
	for key, value := range data.Meta {
		meta[key] = value
	}
	if len(meta) == 0 {
		meta = nil
	}

	text := data.Text
	if text == "" && data.StructuredContent != nil {
		text = compactJSON(data.StructuredContent)
	}
	content := []ContentItem{}
	if text != "" {
		content = append(content, ContentItem{Type: "text", Text: text})
	}

	return CallToolResult{
		Content:           content,
		StructuredContent: data.StructuredContent,
		Meta:              meta,
		IsError:           data.IsError,
	}
}

func authMetaFromContext(ctx context.Context) map[string]any {
	authState := RequestAuthFromContext(ctx)
	if authState.Challenge == "" {
		return nil
	}

	return map[string]any{
		"mcp/www_authenticate": []string{authState.Challenge},
	}
}

func compactJSON(value interface{}) string {
	if value == nil {
		return ""
	}

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	return string(data)
}
