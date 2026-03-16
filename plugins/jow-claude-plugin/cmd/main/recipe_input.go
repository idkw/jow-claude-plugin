package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/idkw/jow-claude-plugin/pkg/jow"
)

type constituentInput struct {
	IngredientID     string  `json:"ingredient_id"`
	QuantityPerCover float64 `json:"quantity_per_cover"`
	UnitID           string  `json:"unit_id"`
}

type toolInput struct {
	ToolID string `json:"tool_id"`
}

// buildRecipeFromArgs parses and resolves all recipe fields from MCP arguments.
func buildRecipeFromArgs(args map[string]interface{}, client *jow.Client) (*jow.Recipe, error) {
	cookingTime := intArg(args, "cooking_time_minutes")
	constituentsJSON, _ := args["constituents"].(string)
	description, _ := args["description"].(string)
	directionsJSON, _ := args["directions"].(string)
	imageURL, _ := args["image_url"].(string)
	preparationTime := intArg(args, "preparation_time_minutes")
	recipeFamily, _ := args["recipe_family"].(string)
	restingTime := optIntArg(args, "resting_time_minutes")
	servings := intArg(args, "servings")
	staticServings, _ := args["static_servings"].(bool)
	tip, _ := args["tip"].(string)
	title, _ := args["title"].(string)
	toolsJSON, _ := args["tools"].(string)

	var rawConstituents []constituentInput
	if err := json.Unmarshal([]byte(constituentsJSON), &rawConstituents); err != nil {
		return nil, fmt.Errorf("invalid constituents JSON: %v", err)
	}
	constituents, err := resolveConstituents(client, rawConstituents)
	if err != nil {
		return nil, fmt.Errorf("resolve constituents: %v", err)
	}

	mainConstituents := make([]jow.Constituent, 0, len(rawConstituents))
	additionalConstituents := make([]jow.Constituent, 0, len(rawConstituents))
	for _, c := range constituents {
		if c.Ingredient.IsAdditionalConstituent {
			additionalConstituents = append(additionalConstituents, c)
		} else {
			mainConstituents = append(mainConstituents, c)
		}
	}

	var steps []string
	if err := json.Unmarshal([]byte(directionsJSON), &steps); err != nil {
		return nil, fmt.Errorf("invalid directions JSON: %v", err)
	}
	directions := make([]jow.Direction, 0, len(steps))
	for _, step := range steps {
		directions = append(directions, jow.Direction{
			Label:               step,
			InvolvedIngredients: []jow.Ingredient{},
		})
	}

	var rawTools []toolInput
	if err := json.Unmarshal([]byte(toolsJSON), &rawTools); err != nil {
		return nil, fmt.Errorf("invalid tools JSON: %v", err)
	}
	tools, err := resolveTools(client, rawTools)
	if err != nil {
		return nil, fmt.Errorf("resolve tools: %v", err)
	}

	recipe := &jow.Recipe{
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
		StaticCoversCount: staticServings,
		Tip:               jow.Tip{Description: tip},
		Title:             title,
		UserConstituents:  []jow.Constituent{},
		UserCoversCount:   servings,
	}
	return recipe, nil
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
		return tools, fmt.Errorf("failed to get all recipe tools: %w", err)
	}
	allToolsByID := make(map[string]jow.Tool)
	for _, t := range allTools {
		allToolsByID[t.ID] = t
	}
	for _, inp := range inputs {
		t, ok := allToolsByID[inp.ToolID]
		if !ok {
			return nil, fmt.Errorf("unknown recipe tool id %v", inp.ToolID)
		}
		tools = append(tools, t)
	}
	return tools, nil
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
