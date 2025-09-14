package api

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/kazegusuri/claude-companion/internal/logger"
	"github.com/kazegusuri/claude-companion/internal/server/db"
	"github.com/kazegusuri/claude-companion/internal/server/handler"
)

// APIHandler manages HTTP API endpoints
type APIHandler struct {
	database       *db.DB
	sessionManager handler.SessionGetter
}

// NewHandler creates a new API handler
func NewHandler(database *db.DB, sessionManager handler.SessionGetter) *APIHandler {
	return &APIHandler{
		database:       database,
		sessionManager: sessionManager,
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
		agentResponse := Agent{
			Id:          agent.ID,
			Pid:         int32(agent.PID),
			SessionId:   agent.SessionID,
			ProjectDir:  agent.ProjectDir,
			ProjectName: extractProjectName(agent.ProjectDir),
			AgentType:   agent.AgentType,
			CreatedAt:   agent.CreatedAt,
			UpdatedAt:   agent.UpdatedAt,
			Session:     h.convertSessionToAPI(agent.SessionID),
		}
		response.Agents = append(response.Agents, agentResponse)
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
		Session:     h.convertSessionToAPI(agent.SessionID),
	}, nil
}

// CreateEchoStrictHandler creates a new echo handler with strict type checking
func CreateEchoStrictHandler(database *db.DB, sessionManager handler.SessionGetter) ServerInterface {
	handler := NewHandler(database, sessionManager)
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

// convertSessionToAPI converts internal Session to API Session model
func (h *APIHandler) convertSessionToAPI(sessionID string) *Session {
	if h.sessionManager == nil {
		return nil
	}

	session, exists := h.sessionManager.GetSession(sessionID)
	if !exists {
		return nil
	}

	// Convert active tool
	var activeTool *ToolInfo
	if tool := session.GetActiveTool(); tool != nil {
		activeTool = &ToolInfo{
			ToolUseId:         tool.ToolUseID,
			ToolName:          tool.ToolName,
			Status:            ToolStatus(tool.Status),
			CreatedAt:         tool.CreatedAt,
			UpdatedAt:         tool.UpdatedAt,
			IsError:           tool.IsError,
			IsRejected:        tool.IsRejected,
			IsWaitingApproval: tool.IsWaitingApproval,
		}
	}

	// Convert background tasks
	var backgroundTasks []BackgroundTaskInfo
	for _, bgTask := range session.GetBackgroundTasks() {
		apiTask := BackgroundTaskInfo{
			BackgroundTaskId: bgTask.BackgroundTaskID,
			ToolUseId:        bgTask.ToolUseID,
			Command:          bgTask.Command,
			Description:      bgTask.Description,
			IsTerminated:     bgTask.IsTerminated,
			CreatedAt:        bgTask.CreatedAt,
		}
		if bgTask.TerminatedAt != nil {
			terminatedAt := *bgTask.TerminatedAt
			apiTask.TerminatedAt = &terminatedAt
		}
		backgroundTasks = append(backgroundTasks, apiTask)
	}

	// Convert to API Session
	apiSession := &Session{
		SessionId:      session.SessionID,
		Uuid:           session.UUID,
		Cwd:            session.CWD,
		TranscriptPath: session.TranscriptPath,
		StartTime:      session.StartTime,
		ActiveTool:     activeTool,
	}

	if len(backgroundTasks) > 0 {
		apiSession.BackgroundTasks = &backgroundTasks
	}

	return apiSession
}
