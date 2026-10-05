package controller

import (
	"agentPribadi/mcp/tools"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func Run() error {
	s := server.NewMCPServer("qtera-inventory-mcp", "2.0.0", server.WithToolCapabilities(true))

	s.AddTools(
		server.ServerTool{
			Tool: mcp.NewTool("bash",
				mcp.WithDescription("Execute bash command"),
				mcp.WithString("command", mcp.Required(), mcp.Description("Command to run")),
			),
			Handler: tools.BashHandler, // no parentheses
		})

	return server.ServeStdio(s)
}
