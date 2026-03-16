package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/idkw/jow-claude-plugin/pkg/jow"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerCatalogTools(s *server.MCPServer, client *jow.Client) {
	registerSearchIngredientsTool(s, client)
	registerGetRecipeToolsTool(s, client)
}

func registerSearchIngredientsTool(s *server.MCPServer, client *jow.Client) {
	s.AddTool(
		mcp.NewTool("search_ingredients",
			mcp.WithDescription(
				"Search Jow's ingredient catalog by name. "+
					"Returns matching ingredients with their IDs and all available measurement units (natural + alternatives). "+
					"IMPORTANT: inspect the returned units carefully and pick the unit_id that matches "+
					"the unit written in the source recipe (e.g. Cuillere a soupe, Pincee, Kilogramme). "+
					"Pass that unit_id to create_recipe — never assume the natural unit is correct.",
			),
			mcp.WithString("query",
				mcp.Required(),
				mcp.Description("Ingredient name to search (French preferred, e.g. 'poulet', 'tomate', 'udon')"),
			),
			mcp.WithNumber("limit",
				mcp.Description("Max results to return (default 5)"),
			),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			args := req.GetArguments()
			query, _ := args["query"].(string)
			limit := 5
			if l, ok := args["limit"].(float64); ok && l > 0 {
				limit = int(l)
			}

			ingredients, err := client.SearchIngredients(query, limit)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("search failed: %v", err)), nil
			}

			type unitInfo struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			type result struct {
				ID               string     `json:"id"`
				Name             string     `json:"name"`
				NaturalUnit      unitInfo   `json:"natural_unit"`
				AlternativeUnits []unitInfo `json:"alternative_units,omitempty"`
			}

			out := make([]result, 0, len(ingredients))
			for _, ing := range ingredients {
				r := result{
					ID:   ing.ID,
					Name: ing.Name,
					NaturalUnit: unitInfo{
						ID:   ing.NaturalUnit.ID,
						Name: ing.NaturalUnit.Name,
					},
				}
				for _, au := range ing.AlternativeUnits {
					r.AlternativeUnits = append(r.AlternativeUnits, unitInfo{
						ID:   au.Unit.ID,
						Name: au.Unit.Name,
					})
				}
				for _, du := range ing.DisplayableUnits {
					r.AlternativeUnits = append(r.AlternativeUnits, unitInfo{
						ID:   du.Unit.ID,
						Name: du.Unit.Name,
					})
				}
				out = append(out, r)
			}

			data, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}

func registerGetRecipeToolsTool(s *server.MCPServer, client *jow.Client) {
	s.AddTool(
		mcp.NewTool("get_recipe_tools",
			mcp.WithDescription(
				"Get the list of kitchen tools available on Jow (wok, casserole, four, etc.). "+
					"Use the tool IDs when creating a recipe.",
			),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			tools, err := client.GetRecipeTools()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get tools failed: %v", err)), nil
			}

			type toolInfo struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}
			out := make([]toolInfo, 0, len(tools))
			for _, t := range tools {
				out = append(out, toolInfo{ID: t.ID, Name: t.Name})
			}

			data, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}
