// Package mcp implements the Model Context Protocol server for Associate.
// MCP allows LLM agents to interact with the graph memory system.
package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fitz/associate/internal/graph"
)

// JSONRPCRequest represents a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

// RPCError represents a JSON-RPC 2.0 error.
type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Server represents an MCP server instance.
type Server struct {
	graphClient *graph.Client
	repoPath    string
	reader      *bufio.Reader
	writer      io.Writer
}

// NewServer creates a new MCP server.
func NewServer(graphClient *graph.Client, repoPath string) *Server {
	return &Server{
		graphClient: graphClient,
		repoPath:    repoPath,
		reader:      bufio.NewReader(os.Stdin),
		writer:      os.Stdout,
	}
}

// Start starts the MCP server and processes requests from stdin.
func (s *Server) Start(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Read a line from stdin
			line, err := s.reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return fmt.Errorf("failed to read request: %w", err)
			}

			// Parse JSON-RPC request
			var req JSONRPCRequest
			if err := json.Unmarshal(line, &req); err != nil {
				s.sendError(nil, -32700, "Parse error", err.Error())
				continue
			}

			// Handle request
			s.handleRequest(ctx, &req)
		}
	}
}

// handleRequest processes a single JSON-RPC request.
func (s *Server) handleRequest(ctx context.Context, req *JSONRPCRequest) {
	switch req.Method {
	case "initialize":
		s.handleInitialize(req)
	case "tools/list":
		s.handleToolsList(req)
	case "tools/call":
		s.handleToolsCall(ctx, req)
	default:
		s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

// handleInitialize handles the initialize request.
func (s *Server) handleInitialize(req *JSONRPCRequest) {
	result := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]bool{},
		},
		"serverInfo": map[string]string{
			"name":    "associate-mcp",
			"version": "0.1.0",
		},
	}
	s.sendResponse(req.ID, result)
}

// handleToolsList handles the tools/list request.
func (s *Server) handleToolsList(req *JSONRPCRequest) {
	tools := []map[string]interface{}{
		{
			"name":        "save_memory",
			"description": "Save a memory or context note to the graph database for the current repository",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"content": map[string]interface{}{
						"type":        "string",
						"description": "The memory content to save",
					},
					"context_type": map[string]interface{}{
						"type":        "string",
						"description": "Type of context: architectural_decision, bug_fix, performance, etc.",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Tags for categorization (optional)",
					},
					"related_to": map[string]interface{}{
						"type":        "string",
						"description": "Related code path (optional)",
					},
				},
				"required": []string{"content", "context_type"},
			},
		},
		{
			"name":        "search_memory",
			"description": "Search for memories in the current repository",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (optional)",
					},
					"context_type": map[string]interface{}{
						"type":        "string",
						"description": "Filter by context type (optional)",
					},
					"tags": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Filter by tags (optional)",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Maximum results to return (default: 10)",
					},
				},
			},
		},
		{
			"name":        "save_learning",
			"description": "Save an architectural pattern or learning specific to the current repository",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "The pattern or learning",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Category: architectural_pattern, best_practice, anti_pattern, etc.",
					},
					"description": map[string]interface{}{
						"type":        "string",
						"description": "Detailed description",
					},
					"examples": map[string]interface{}{
						"type":        "array",
						"items":       map[string]string{"type": "string"},
						"description": "Code examples (optional)",
					},
				},
				"required": []string{"pattern", "category", "description"},
			},
		},
		{
			"name":        "search_learnings",
			"description": "Search for architectural patterns and learnings in the current repository",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Search query (optional)",
					},
					"category": map[string]interface{}{
						"type":        "string",
						"description": "Filter by category (optional)",
					},
					"limit": map[string]interface{}{
						"type":        "number",
						"description": "Maximum results to return (default: 10)",
					},
				},
			},
		},
		{
			"name":        "get_repo_context",
			"description": "Get repository-specific context like AGENTS.md if it exists",
			"inputSchema": map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	}

	result := map[string]interface{}{
		"tools": tools,
	}
	s.sendResponse(req.ID, result)
}

// handleToolsCall handles the tools/call request.
func (s *Server) handleToolsCall(ctx context.Context, req *JSONRPCRequest) {
	var params struct {
		Name      string                 `json:"name"`
		Arguments map[string]interface{} `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		s.sendError(req.ID, -32602, "Invalid params", err.Error())
		return
	}

	switch params.Name {
	case "save_memory":
		s.toolSaveMemory(ctx, req.ID, params.Arguments)
	case "search_memory":
		s.toolSearchMemory(ctx, req.ID, params.Arguments)
	case "save_learning":
		s.toolSaveLearning(ctx, req.ID, params.Arguments)
	case "search_learnings":
		s.toolSearchLearnings(ctx, req.ID, params.Arguments)
	case "get_repo_context":
		s.toolGetRepoContext(req.ID)
	default:
		s.sendError(req.ID, -32602, "Unknown tool", params.Name)
	}
}

// toolSaveMemory implements the save_memory tool.
func (s *Server) toolSaveMemory(ctx context.Context, id interface{}, args map[string]interface{}) {
	content, _ := args["content"].(string)
	contextType, _ := args["context_type"].(string)
	relatedTo, _ := args["related_to"].(string)

	var tags []string
	if tagsList, ok := args["tags"].([]interface{}); ok {
		for _, t := range tagsList {
			if tag, ok := t.(string); ok {
				tags = append(tags, tag)
			}
		}
	}

	memory := &graph.MemoryNode{
		Content:     content,
		ContextType: contextType,
		Tags:        tags,
		RelatedTo:   relatedTo,
	}

	if err := s.graphClient.SaveMemory(ctx, s.repoPath, memory); err != nil {
		s.sendError(id, -32603, "Failed to save memory", err.Error())
		return
	}

	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("✓ Memory saved successfully\nType: %s\nTags: %v", contextType, tags),
			},
		},
	}
	s.sendResponse(id, result)
}

// toolSearchMemory implements the search_memory tool.
func (s *Server) toolSearchMemory(ctx context.Context, id interface{}, args map[string]interface{}) {
	query, _ := args["query"].(string)
	contextType, _ := args["context_type"].(string)
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	var tags []string
	if tagsList, ok := args["tags"].([]interface{}); ok {
		for _, t := range tagsList {
			if tag, ok := t.(string); ok {
				tags = append(tags, tag)
			}
		}
	}

	memories, err := s.graphClient.SearchMemory(ctx, s.repoPath, query, contextType, tags, limit)
	if err != nil {
		s.sendError(id, -32603, "Failed to search memory", err.Error())
		return
	}

	var textResult string
	if len(memories) == 0 {
		textResult = "No memories found matching the criteria."
	} else {
		textResult = fmt.Sprintf("Found %d memory(ies):\n\n", len(memories))
		for i, m := range memories {
			textResult += fmt.Sprintf("%d. [%s] %s\n", i+1, m.ContextType, m.Content)
			if len(m.Tags) > 0 {
				textResult += fmt.Sprintf("   Tags: %v\n", m.Tags)
			}
			if m.RelatedTo != "" {
				textResult += fmt.Sprintf("   Related: %s\n", m.RelatedTo)
			}
			textResult += "\n"
		}
	}

	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": textResult,
			},
		},
	}
	s.sendResponse(id, result)
}

// toolSaveLearning implements the save_learning tool.
func (s *Server) toolSaveLearning(ctx context.Context, id interface{}, args map[string]interface{}) {
	pattern, _ := args["pattern"].(string)
	category, _ := args["category"].(string)
	description, _ := args["description"].(string)

	var examples []string
	if examplesList, ok := args["examples"].([]interface{}); ok {
		for _, e := range examplesList {
			if example, ok := e.(string); ok {
				examples = append(examples, example)
			}
		}
	}

	learning := &graph.LearningNode{
		Pattern:     pattern,
		Category:    category,
		Description: description,
		Examples:    examples,
	}

	if err := s.graphClient.SaveLearning(ctx, s.repoPath, learning); err != nil {
		s.sendError(id, -32603, "Failed to save learning", err.Error())
		return
	}

	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": fmt.Sprintf("✓ Learning saved successfully\nPattern: %s\nCategory: %s", pattern, category),
			},
		},
	}
	s.sendResponse(id, result)
}

// toolSearchLearnings implements the search_learnings tool.
func (s *Server) toolSearchLearnings(ctx context.Context, id interface{}, args map[string]interface{}) {
	query, _ := args["query"].(string)
	category, _ := args["category"].(string)
	limit := 10
	if l, ok := args["limit"].(float64); ok {
		limit = int(l)
	}

	learnings, err := s.graphClient.SearchLearnings(ctx, s.repoPath, query, category, limit)
	if err != nil {
		s.sendError(id, -32603, "Failed to search learnings", err.Error())
		return
	}

	var textResult string
	if len(learnings) == 0 {
		textResult = "No learnings found matching the criteria."
	} else {
		textResult = fmt.Sprintf("Found %d learning(s):\n\n", len(learnings))
		for i, l := range learnings {
			textResult += fmt.Sprintf("%d. [%s] %s\n", i+1, l.Category, l.Pattern)
			textResult += fmt.Sprintf("   Description: %s\n", l.Description)
			if len(l.Examples) > 0 {
				textResult += fmt.Sprintf("   Examples: %d provided\n", len(l.Examples))
			}
			textResult += "\n"
		}
	}

	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": textResult,
			},
		},
	}
	s.sendResponse(id, result)
}

// toolGetRepoContext implements the get_repo_context tool.
func (s *Server) toolGetRepoContext(id interface{}) {
	// Check for AGENTS.md in the repo
	agentsPath := filepath.Join(s.repoPath, "AGENTS.md")
	content, err := os.ReadFile(agentsPath)
	
	var textResult string
	if err != nil {
		if os.IsNotExist(err) {
			textResult = "No AGENTS.md file found in this repository."
		} else {
			textResult = fmt.Sprintf("Error reading AGENTS.md: %v", err)
		}
	} else {
		textResult = fmt.Sprintf("=== Repository Context (AGENTS.md) ===\n\n%s", string(content))
	}

	result := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": textResult,
			},
		},
	}
	s.sendResponse(id, result)
}

// sendResponse sends a successful JSON-RPC response.
func (s *Server) sendResponse(id interface{}, result interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.writeResponse(&resp)
}

// sendError sends a JSON-RPC error response.
func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.writeResponse(&resp)
}

// writeResponse writes a JSON-RPC response to stdout.
func (s *Server) writeResponse(resp *JSONRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to marshal response: %v\n", err)
		return
	}
	data = append(data, '\n')
	if _, err := s.writer.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to write response: %v\n", err)
	}
}
