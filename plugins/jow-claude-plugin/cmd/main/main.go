package main

import (
	"log"
	"os"

	"github.com/idkw/jow-claude-plugin/pkg/jow"

	"github.com/mark3labs/mcp-go/server"
)

func main() {
	token := os.Getenv("JOW_TOKEN")
	if token == "" {
		log.Fatal("JOW_TOKEN environment variable not set")
	}
	client := jow.NewClient(token)

	s := server.NewMCPServer("jow-recipes", "1.0.0",
		server.WithToolCapabilities(true),
	)

	registerTools(s, client)

	if addr := os.Getenv("JOW_MCP_HTTP_ADDR"); addr != "" {
		log.Printf("Started StreamableHTTP MCP sever listening on %s", addr)
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(addr); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	} else {
		log.Printf("Started stdio MCP server")
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	}
}

func registerTools(s *server.MCPServer, client *jow.Client) {
	registerCatalogTools(s, client)
	registerRecipeTools(s, client)
	registerCollectionTools(s, client)
}
