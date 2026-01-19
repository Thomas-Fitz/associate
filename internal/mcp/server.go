package mcp

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/fitz/associate/internal/mcp/tools"
	"github.com/fitz/associate/internal/neo4j"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	ServerName    = "associate"
	ServerVersion = "v1.0.0"
)

// Server wraps the MCP server with Associate-specific configuration
type Server struct {
	mcpServer *mcp.Server
	repo      *neo4j.Repository
	planRepo  *neo4j.PlanRepository
	taskRepo  *neo4j.TaskRepository
	logger    *slog.Logger
	handler   *tools.Handler
}

// NewServer creates a new Associate MCP server
func NewServer(repo *neo4j.Repository, planRepo *neo4j.PlanRepository, taskRepo *neo4j.TaskRepository, logger *slog.Logger) *Server {
	if logger == nil {
		logger = slog.Default()
	}

	mcpServer := mcp.NewServer(
		&mcp.Implementation{
			Name:    ServerName,
			Version: ServerVersion,
		},
		nil,
	)

	s := &Server{
		mcpServer: mcpServer,
		repo:      repo,
		planRepo:  planRepo,
		taskRepo:  taskRepo,
		logger:    logger,
		handler:   tools.NewHandler(repo, planRepo, taskRepo, logger),
	}

	s.registerTools()
	return s
}

// registerTools adds all MCP tools to the server
func (s *Server) registerTools() {
	// Memory tools
	mcp.AddTool(s.mcpServer, tools.SearchTool(), s.handler.HandleSearch)
	mcp.AddTool(s.mcpServer, tools.GetTool(), s.handler.HandleGet)
	mcp.AddTool(s.mcpServer, tools.AddTool(), s.handler.HandleAdd)
	mcp.AddTool(s.mcpServer, tools.UpdateTool(), s.handler.HandleUpdate)
	mcp.AddTool(s.mcpServer, tools.DeleteTool(), s.handler.HandleDelete)
	mcp.AddTool(s.mcpServer, tools.GetRelatedTool(), s.handler.HandleGetRelated)

	// Plan tools
	mcp.AddTool(s.mcpServer, tools.CreatePlanTool(), s.handler.HandleCreatePlan)
	mcp.AddTool(s.mcpServer, tools.GetPlanTool(), s.handler.HandleGetPlan)
	mcp.AddTool(s.mcpServer, tools.UpdatePlanTool(), s.handler.HandleUpdatePlan)
	mcp.AddTool(s.mcpServer, tools.DeletePlanTool(), s.handler.HandleDeletePlan)
	mcp.AddTool(s.mcpServer, tools.ListPlansTool(), s.handler.HandleListPlans)

	// Task tools
	mcp.AddTool(s.mcpServer, tools.CreateTaskTool(), s.handler.HandleCreateTask)
	mcp.AddTool(s.mcpServer, tools.GetTaskTool(), s.handler.HandleGetTask)
	mcp.AddTool(s.mcpServer, tools.UpdateTaskTool(), s.handler.HandleUpdateTask)
	mcp.AddTool(s.mcpServer, tools.DeleteTaskTool(), s.handler.HandleDeleteTask)
	mcp.AddTool(s.mcpServer, tools.ListTasksTool(), s.handler.HandleListTasks)
}

// HTTPHandler returns an http.Handler for the MCP server
func (s *Server) HTTPHandler() http.Handler {
	return mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			return s.mcpServer
		},
		&mcp.StreamableHTTPOptions{
			Logger: s.logger,
		},
	)
}

// Run starts the MCP server over stdio (for CLI usage)
func (s *Server) Run(ctx context.Context) error {
	return s.mcpServer.Run(ctx, &mcp.StdioTransport{})
}
