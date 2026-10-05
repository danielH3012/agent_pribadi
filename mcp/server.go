package mcp

import (
	"github.com/mark3labs/mcp-go/server"

	"agentPribadi/mcp/client"
	//"agentPribadi/mcp/controller"
)

func main() {
	client.InitLogger()

	s := server.NewMCPServer(
		"qtera-inventory-mcp",
		"2.0.0",
		server.WithToolCapabilities(true),
		server.WithLogging(),
	)

	if err := server.ServeStdio(s); err != nil {
		if client.Logger != nil {
			client.Logger.Fatalf("server error: %v", err)
		}
	}
}
