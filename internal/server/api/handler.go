package api

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/server/db"
)

// APIHandler manages HTTP API endpoints
type APIHandler struct {
	database *db.DB
}

// NewHandler creates a new API handler
func NewHandler(database *db.DB) *APIHandler {
	return &APIHandler{
		database: database,
	}
}

// Ensure APIHandler implements the generated StrictServerInterface
var _ StrictServerInterface = (*APIHandler)(nil)

// AgentsList implements GET /api/agents
func (h *APIHandler) AgentsList(ctx context.Context, request AgentsListRequestObject) (AgentsListResponseObject, error) {
	// Get agents from database
	agents, err := h.database.ListClaudeAgents()
	if err != nil {
		logger.LogError("Failed to list agents: %v", err)
		return nil, err
	}

	// Convert to response format
	response := AgentListResponse{
		Agents: make([]Agent, 0, len(agents)),
	}

	for _, agent := range agents {
		response.Agents = append(response.Agents, Agent{
			Id:          agent.ID,
			Pid:         int32(agent.PID),
			SessionId:   agent.SessionID,
			ProjectDir:  agent.ProjectDir,
			ProjectName: extractProjectName(agent.ProjectDir),
			AgentType:   agent.AgentType,
			CreatedAt:   agent.CreatedAt,
			UpdatedAt:   agent.UpdatedAt,
		})
	}

	return AgentsList200JSONResponse(response), nil
}

// AgentsRead implements GET /api/agents/{id}
func (h *APIHandler) AgentsRead(ctx context.Context, request AgentsReadRequestObject) (AgentsReadResponseObject, error) {
	// Get agent from database by ID
	agent, err := h.database.GetClaudeAgentByID(request.Id)
	if err != nil {
		logger.LogError("Failed to get agent: %v", err)
		return nil, err
	}

	if agent == nil {
		return AgentsRead404JSONResponse{
			Message: "Agent not found",
		}, nil
	}

	return AgentsRead200JSONResponse{
		Id:          agent.ID,
		Pid:         int32(agent.PID),
		SessionId:   agent.SessionID,
		ProjectDir:  agent.ProjectDir,
		ProjectName: extractProjectName(agent.ProjectDir),
		AgentType:   agent.AgentType,
		CreatedAt:   agent.CreatedAt,
		UpdatedAt:   agent.UpdatedAt,
	}, nil
}

// CreateEchoStrictHandler creates a new echo handler with strict type checking
func CreateEchoStrictHandler(database *db.DB) ServerInterface {
	handler := NewHandler(database)
	return NewStrictHandler(handler, nil)
}

// extractProjectName extracts the project name from the project directory path
func extractProjectName(projectDir string) string {
	// Remove trailing slashes
	projectDir = strings.TrimSuffix(projectDir, "/")

	// Get the base name (last component of the path)
	baseName := filepath.Base(projectDir)

	// If the base name is empty or ".", return the directory itself
	if baseName == "" || baseName == "." {
		return projectDir
	}

	return baseName
}
