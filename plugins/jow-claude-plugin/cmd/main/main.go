package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/idkw/jow-claude-plugin/pkg/jow"

	"github.com/mark3labs/mcp-go/mcp"
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
		httpServer := server.NewStreamableHTTPServer(s)
		if err := httpServer.Start(addr); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	} else {
		if err := server.ServeStdio(s); err != nil {
			log.Fatalf("MCP server error: %v", err)
		}
	}
}

func registerTools(s *server.MCPServer, client *jow.Client) {
	// ── search_ingredients ──────────────────────────────────────────────────
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

	// ── get_recipe_tools ────────────────────────────────────────────────────
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

	// ── upload_recipe_image ─────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("upload_recipe_image",
			mcp.WithDescription(
				"Upload a local image file as the picture for a Jow recipe. "+
					"Returns the imageUrl to pass to set_recipe_image or update_recipe.",
			),
			mcp.WithString("file_path",
				mcp.Required(),
				mcp.Description("Absolute path to the image file on disk (JPEG, PNG, …)"),
			),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(false),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			filePath, _ := req.GetArguments()["file_path"].(string)
			imageURL, err := client.UploadRecipeImage(filePath)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("upload image: %v", err)), nil
			}
			return mcp.NewToolResultText(imageURL), nil
		},
	)

	// ── create_recipe ───────────────────────────────────────────────────────
	s.AddTool(
		mcp.NewTool("create_recipe",
			mcp.WithDescription(`Create a complete recipe on Jow.

Workflow:
1. Call search_ingredients for each ingredient to obtain its id and unit id.
2. Call get_recipe_tools for each tool to obtain its id
3. Upload the recipe image if provided by the user
4. Call create_recipe with all the recipe data.

constituents : all ingredients (meat, vegetables, pasta, salt, butter, ...)

  Each constituent is a JSON object:
    {"ingredient_id": "<id from search_ingredients>", "quantity_per_cover": <number>, "unit_id": "<unit id>"}
    quantity_per_cover = total quantity for the recipe ÷ number of servings (in the chosen unit)
  The ingredient_id and unit_id comes from the search_ingredients tool 

tools: all tools used in the recipe
  Each tool is a JSON object:
    {"tool_id": "<id from search>"}

directions : ordered array of step descriptions as plain strings`),
			mcp.WithString("constituents",
				mcp.Required(),
				mcp.Description(`JSON array of ingredients: [{"ingredient_id":"...","quantity_per_cover":0.2,"unit_id":"..."}]`),
			),
			mcp.WithNumber("cooking_time_minutes", mcp.Required(), mcp.Description("Cooking time in minutes")),
			mcp.WithNumber("description", mcp.Required(), mcp.Description("Short description about the recipe")),
			mcp.WithString("directions",
				mcp.Required(),
				mcp.Description(`JSON array of step strings: ["Faire bouillir l'eau...", "Ajouter les pâtes..."]`),
			),
			mcp.WithString("image_url",
				mcp.Description(`Path of the image previously uploaded to jow using the upload_recipe_image tool (e.g "uploadedrecipes/0pShp2tcyOcmtQ.jpg")`),
			),
			mcp.WithNumber("preparation_time_minutes", mcp.Required(), mcp.Description("Preparation time in minutes")),
			mcp.WithString("recipe_family", mcp.Required(), mcp.Description("Type of recipe"), mcp.Enum("Plat", "Dessert", "Apéro", "Boisson", "Entrée", "Autre")),
			mcp.WithNumber("resting_time_minutes", mcp.Description("Resting time in minutes")),
			mcp.WithNumber("servings", mcp.Required(), mcp.Description("Number of servings. Set 0 if this recipe is for sharing and can't be cooked as individual servings")),
			mcp.WithString("tip",
				mcp.Description("Optional chef tip shown with the recipe"),
			),
			mcp.WithString("title", mcp.Required(), mcp.Description("Recipe title")),
			mcp.WithString("tools",
				mcp.Required(),
				mcp.Description(`JSON array of tools: [{"tool_id":"..."}]`),
			),
			mcp.WithDestructiveHintAnnotation(true),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			return handleCreateRecipe(req, client)
		},
	)
}

// ── create_recipe handler ────────────────────────────────────────────────────

type constituentInput struct {
	IngredientID     string  `json:"ingredient_id"`
	QuantityPerCover float64 `json:"quantity_per_cover"`
	UnitID           string  `json:"unit_id"`
}
type toolInput struct {
	ToolID string `json:"tool_id"`
}

func handleCreateRecipe(req mcp.CallToolRequest, client *jow.Client) (*mcp.CallToolResult, error) {
	args := req.GetArguments()

	cookingTime := intArg(args, "cooking_time_minutes")
	constituentsJSON, _ := args["constituents"].(string)
	description, _ := args["description"].(string)
	directionsJSON, _ := args["directions"].(string)
	imageURL, _ := args["image_url"].(string)
	preparationTime := intArg(args, "preparation_time_minutes")
	recipeFamily, _ := args["recipe_family"].(string)
	restingTime := optIntArg(args, "resting_time_minutes")
	servings := intArg(args, "servings")
	staticCoversCount := false
	if servings == 0 {
		staticCoversCount = true
	}
	tip, _ := args["tip"].(string)
	title, _ := args["title"].(string)
	toolsJSON, _ := args["tools"].(string)

	// Parse and resolve constituents
	var rawConstituents []constituentInput
	if err := json.Unmarshal([]byte(constituentsJSON), &rawConstituents); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid constituents JSON: %v", err)), nil
	}
	constituents, err := resolveConstituents(client, rawConstituents)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolve constituents: %v", err)), nil
	}

	mainConstituents := make([]jow.Constituent, 0, len(rawConstituents))
	additionalConstituents := make([]jow.Constituent, 0, len(rawConstituents))
	for _, c := range constituents {
		if c.Ingredient.RecipeUploadConfig.IsAdditionalConstituent {
			additionalConstituents = append(additionalConstituents, c)
		} else {
			mainConstituents = append(mainConstituents, c)
		}
	}

	// Parse directions
	var steps []string
	if err := json.Unmarshal([]byte(directionsJSON), &steps); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid directions JSON: %v", err)), nil
	}
	directions := make([]jow.Direction, 0, len(steps))
	for _, step := range steps {
		directions = append(directions, jow.Direction{
			Label:               step,
			InvolvedIngredients: []jow.Ingredient{},
		})
	}

	// Parse and resolve tools
	var rawTools []toolInput
	if err := json.Unmarshal([]byte(toolsJSON), &rawTools); err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("invalid tools JSON: %v", err)), nil
	}
	tools, err := resolveTools(client, rawTools)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("resolve tools: %v", err)), nil
	}

	recipeReq := jow.Recipe{
		AdditionalConstituents: additionalConstituents,
		BackgroundPattern: jow.BackgroundPattern{
			Color:    "#ffc847",
			ImageUrl: "patterns/yolk-01.png",
		},
		Constituents:      mainConstituents,
		CookingTime:       cookingTime,
		Description:       description,
		Directions:        directions,
		ImageURL:          imageURL,
		PlaceHolderURL:    "placeholders/plate.png",
		PreparationTime:   preparationTime,
		RecipeFamily:      resolveRecipeFamily(recipeFamily),
		RestingTime:       restingTime,
		RequiredTools:     tools,
		StaticCoversCount: staticCoversCount,
		Tip:               jow.Tip{Description: tip},
		Title:             title,
		UserConstituents:  []jow.Constituent{},
		UserCoversCount:   servings,
	}

	err = client.CreateRecipe(recipeReq)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("create recipe: %v", err)), nil
	}

	recipe, err := client.GetMostRecentRecipeByTitle(recipeReq.Title)
	if err != nil {
		return mcp.NewToolResultError(fmt.Sprintf("get most recent recipe: %v", err)), nil
	}

	recipeURL := fmt.Sprintf("https://jow.fr/user-recipes/%v", recipe.ID)
	return mcp.NewToolResultText(fmt.Sprintf("Recipe created successfully! ID: %v URL: %v", recipe.ID, recipeURL)), nil
}

func resolveRecipeFamily(input string) string {
	switch strings.ToLower(input) {
	case "plat":
		return "5fc78542aaaddb03d10f47bc"
	case "dessert":
		return "5fc78569aaaddb03d10f47be"
	case "apéro":
		return "5fc785597324b103d7111a70"
	case "boisson":
		return "5fc785797324b103d7111a73"
	case "entrée":
		return "633c3b219b5b7b27c90771b5"
	case "autre":
		fallthrough
	default:
		return "633c3b0c9b5b7b27c90771a7"
	}
}

// resolveConstituents fetches full ingredient objects for each constituent input.
func resolveConstituents(client *jow.Client, inputs []constituentInput) ([]jow.Constituent, error) {
	constituents := make([]jow.Constituent, 0, len(inputs))
	for _, inp := range inputs {
		ing, err := client.GetIngredient(inp.IngredientID)
		if err != nil {
			return nil, fmt.Errorf("ingredient %s: %w", inp.IngredientID, err)
		}
		unit := resolveUnit(*ing, inp.UnitID)
		constituents = append(constituents, jow.Constituent{
			Ingredient:       *ing,
			QuantityPerCover: inp.QuantityPerCover,
			Unit:             unit,
		})
	}
	return constituents, nil
}

// resolveUnit finds the unit by ID in the ingredient's available units,
// falling back to the natural unit if not found.
func resolveUnit(ing jow.Ingredient, unitID string) jow.Unit {
	if unitID == "" || unitID == ing.NaturalUnit.ID {
		return ing.NaturalUnit
	}
	for _, au := range ing.AlternativeUnits {
		if au.Unit.ID == unitID {
			return au.Unit
		}
	}
	for _, du := range ing.DisplayableUnits {
		if du.Unit.ID == unitID {
			return du.Unit
		}
	}
	return ing.NaturalUnit
}

// resolveTools fetches full tools for each tool input.
func resolveTools(client *jow.Client, inputs []toolInput) ([]jow.Tool, error) {
	tools := make([]jow.Tool, 0, len(inputs))
	if len(inputs) == 0 {
		return tools, nil
	}
	allTools, err := client.GetRecipeTools()
	if err != nil {
		return tools, fmt.Errorf("faield to get all recipe tools: %w", err)
	}
	allToolsByID := make(map[string]jow.Tool)
	for _, t := range allTools {
		allToolsByID[t.ID] = t
	}
	for _, inp := range inputs {
		t, ok := allToolsByID[inp.ToolID]
		if !ok {
			return nil, fmt.Errorf("Unknown recipe tool id %v", inp.ToolID)
		}
		tools = append(tools, t)
	}
	return tools, nil
}

func floatArg(args map[string]interface{}, key string) float64 {
	v, _ := args[key].(float64)
	return v
}

func intArg(args map[string]interface{}, key string) int {
	v, _ := args[key].(float64)
	return int(v)
}

func optIntArg(args map[string]interface{}, key string) *int {
	v, ok := args[key]
	if !ok {
		return nil
	}
	vCast, _ := v.(float64)
	vInt := int(vCast)
	return &vInt
}
