package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/idkw/jow-claude-plugin/pkg/jow"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func registerRecipeTools(s *server.MCPServer, client *jow.Client) {
	registerGetRecipeTitlesTool(s, client)
	registerGetRecipesTool(s, client)
	registerGetRecipeTool(s, client)
	registerUploadRecipeImageTool(s, client)
	registerCreateRecipeTool(s, client)
	registerUpdateRecipeTool(s, client)
}

func registerGetRecipeTitlesTool(s *server.MCPServer, client *jow.Client) {
	s.AddTool(
		mcp.NewTool("get_recipe_titles",
			mcp.WithDescription(
				"Get all user-created recipes with only their ID and title. "+
					"Use this when you need to find a recipe by name to retrieve its ID, "+
					"without fetching full recipe data.",
			),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			recipes, err := client.GetRecipes()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get recipes: %v", err)), nil
			}

			out := make([]recipeTitle, 0, len(recipes))
			for _, r := range recipes {
				out = append(out, recipeTitle{ID: r.ID, Title: r.Title, CreatedAt: r.CreatedAt})
			}

			data, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}

func registerGetRecipesTool(s *server.MCPServer, client *jow.Client) {
	s.AddTool(
		mcp.NewTool("get_recipes",
			mcp.WithDescription(
				"Get all user-created recipes with full details (ingredients, directions, tools, metadata). "+
					"Use this to retrieve complete recipe data for updating or reviewing recipes.",
			),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			recipes, err := client.GetRecipes()
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get recipes: %v", err)), nil
			}

			out := make([]recipeOutput, 0, len(recipes))
			for _, r := range recipes {
				out = append(out, toRecipeOutput(r))
			}

			data, _ := json.MarshalIndent(out, "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}

func registerGetRecipeTool(s *server.MCPServer, client *jow.Client) {
	s.AddTool(
		mcp.NewTool("get_recipe",
			mcp.WithDescription(
				"Get full details for a single recipe by its ID. "+
					"Use this to retrieve a specific recipe for update or review.",
			),
			mcp.WithString("recipe_id",
				mcp.Required(),
				mcp.Description("ID of the recipe to fetch"),
			),
			mcp.WithDestructiveHintAnnotation(false),
			mcp.WithOpenWorldHintAnnotation(true),
		),
		func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			recipeID, _ := req.GetArguments()["recipe_id"].(string)
			if recipeID == "" {
				return mcp.NewToolResultError("recipe_id is required"), nil
			}

			recipe, err := client.GetRecipeByID(recipeID)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get recipe: %v", err)), nil
			}

			data, _ := json.MarshalIndent(toRecipeOutput(*recipe), "", "  ")
			return mcp.NewToolResultText(string(data)), nil
		},
	)
}

func registerUploadRecipeImageTool(s *server.MCPServer, client *jow.Client) {
	s.AddTool(
		mcp.NewTool("upload_recipe_image",
			mcp.WithDescription(
				"Upload a local image file as the picture for a Jow recipe. "+
					"Returns the imageUrl to pass to create_recipe or update_recipe.",
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
}

func registerCreateRecipeTool(s *server.MCPServer, client *jow.Client) {
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
			mcp.WithString("description", mcp.Required(), mcp.Description("Short description about the recipe")),
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
			mcp.WithNumber("servings", mcp.Required(), mcp.Description("Number of servings.")),
			mcp.WithBoolean("static_servings", mcp.Required(), mcp.Description("Set this to true if this recipe is for sharing and can't be cooked as individual servings.")),
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
			recipe, err := buildRecipeFromArgs(req.GetArguments(), client)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}

			if err := client.CreateRecipe(*recipe); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("create recipe: %v", err)), nil
			}

			created, err := client.GetMostRecentRecipeByTitle(recipe.Title)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("get most recent recipe: %v", err)), nil
			}

			recipeURL := fmt.Sprintf("https://jow.fr/user-recipes/%v", created.ID)
			return mcp.NewToolResultText(fmt.Sprintf("Recipe created successfully! ID: %v URL: %v", created.ID, recipeURL)), nil
		},
	)
}

func registerUpdateRecipeTool(s *server.MCPServer, client *jow.Client) {
	s.AddTool(
		mcp.NewTool("update_recipe",
			mcp.WithDescription(`Update an existing recipe on Jow. Same fields as create_recipe plus the recipe ID.

All fields are replaced — pass the complete recipe data, not just the changed fields.`),
			mcp.WithString("recipe_id",
				mcp.Required(),
				mcp.Description("ID of the recipe to update"),
			),
			mcp.WithString("constituents",
				mcp.Required(),
				mcp.Description(`JSON array of ingredients: [{"ingredient_id":"...","quantity_per_cover":0.2,"unit_id":"..."}]`),
			),
			mcp.WithNumber("cooking_time_minutes", mcp.Required(), mcp.Description("Cooking time in minutes")),
			mcp.WithString("description", mcp.Required(), mcp.Description("Short description about the recipe")),
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
			mcp.WithNumber("servings", mcp.Required(), mcp.Description("Number of servings.")),
			mcp.WithBoolean("static_servings", mcp.Required(), mcp.Description("Set this to true if this recipe is for sharing and can't be cooked as individual servings.")),
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
			args := req.GetArguments()
			recipeID, _ := args["recipe_id"].(string)
			if recipeID == "" {
				return mcp.NewToolResultError("recipe_id is required"), nil
			}

			recipe, err := buildRecipeFromArgs(args, client)
			if err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
			recipe.ID = recipeID

			if err := client.UpdateRecipe(recipeID, *recipe); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("update recipe: %v", err)), nil
			}

			recipeURL := fmt.Sprintf("https://jow.fr/user-recipes/%v", recipeID)
			return mcp.NewToolResultText(fmt.Sprintf("Recipe updated successfully! ID: %v URL: %v", recipeID, recipeURL)), nil

		},
	)
}
