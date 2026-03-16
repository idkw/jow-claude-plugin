package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/idkw/jow-claude-plugin/pkg/jow"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCollectionTools(s *server.MCPServer, client *jow.Client) {
	registerGetCollectionsTool(s, client)
	registerAddRecipeToCollectionTool(s, client)
}

func registerGetCollectionsTool(s *server.MCPServer, client *jow.Client) {
	s.AddTool(
		mcp.NewTool("get_collections",
			mcp.WithDescription(
				"Get the user's Jow collections (favorites, custom lists, …). "+
					"Use the returned IDs with add_recipe_to_collection.",
			),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			collections, err := client.GetCollections()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get collections: %v", err)), nil
			}

			type collectionInfo struct {
				ID          string `json:"id"`
				Title       string `json:"title"`
				Type        string `json:"type"`
				RecipeCount int    `json:"recipe_count"`
			}
			out := make([]collectionInfo, 0, len(collections))
			for _, c := range collections {
				out = append(out, collectionInfo{
					ID:          c.ID,
					Title:       c.Title,
					Type:        c.Type,
					RecipeCount: c.RecipeCount,
				})
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}

func registerAddRecipeToCollectionTool(s *server.MCPServer, client *jow.Client) {
	s.AddTool(
		mcp.NewTool("add_recipe_to_collection",
			mcp.WithDescription(
				"Add a Jow recipe to one or more user collections (e.g. favorites). "+
					"Provide the recipe ID and the collection IDs to add it to. "+
					"Returns the list of updated collections.",
			),
			mcp.WithString("recipe_id",
				mcp.Required(),
				mcp.Description("ID of the recipe to add to collections"),
			),
			mcp.WithString("collection_ids",
				mcp.Required(),
				mcp.Description(`JSON array of collection IDs: ["69b31ae693b871d07e5e201e"]`),
			),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			recipeID, _ := args["recipe_id"].(string)
			collectionIDsJSON, _ := args["collection_ids"].(string)

			var collectionIDs []string
			if err := json.Unmarshal([]byte(collectionIDsJSON), &collectionIDs); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("invalid collection_ids JSON: %v", err)), nil
			}

			updated, err := client.AddRecipeToCollection(recipeID, collectionIDs)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("add to collection: %v", err)), nil
			}

			type collectionInfo struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				Type  string `json:"type"`
			}
			out := make([]collectionInfo, 0, len(updated))
			for _, c := range updated {
				out = append(out, collectionInfo{ID: c.ID, Title: c.Title, Type: c.Type})
			}
			data, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}
