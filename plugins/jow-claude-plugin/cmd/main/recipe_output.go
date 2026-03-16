package main

import (
	"time"

	"github.com/idkw/jow-claude-plugin/pkg/jow"
)

// ── shared output types ──────────────────────────────────────────────────────

type recipeTitle struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	CreatedAt *time.Time `json:"created_at,omitempty"`
}

type unitOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type constituentOutput struct {
	IngredientID   string     `json:"ingredient_id"`
	IngredientName string     `json:"ingredient_name"`
	Quantity       float64    `json:"quantity_per_cover"`
	Unit           unitOutput `json:"unit"`
}

type directionOutput struct {
	Label string `json:"label"`
}

type toolOutput struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type recipeOutput struct {
	ID                     string              `json:"id"`
	Title                  string              `json:"title"`
	Description            string              `json:"description"`
	RecipeFamily           string              `json:"recipe_family"`
	Servings               int                 `json:"servings"`
	StaticServings         bool                `json:"static_servings"`
	PreparationTime        int                 `json:"preparation_time_minutes"`
	CookingTime            int                 `json:"cooking_time_minutes"`
	RestingTime            *int                `json:"resting_time_minutes,omitempty"`
	ImageURL               string              `json:"image_url,omitempty"`
	Tip                    string              `json:"tip,omitempty"`
	Constituents           []constituentOutput `json:"constituents"`
	AdditionalConstituents []constituentOutput `json:"additional_constituents"`
	Directions             []directionOutput   `json:"directions"`
	Tools                  []toolOutput        `json:"tools"`
}

func toConstituentOutputs(cs []jow.Constituent) []constituentOutput {
	out := make([]constituentOutput, 0, len(cs))
	for _, c := range cs {
		out = append(out, constituentOutput{
			IngredientID:   c.Ingredient.ID,
			IngredientName: c.Ingredient.Name,
			Quantity:       c.QuantityPerCover,
			Unit:           unitOutput{ID: c.Unit.ID, Name: c.Unit.Name},
		})
	}
	return out
}

func toRecipeOutput(r jow.Recipe) recipeOutput {
	directions := make([]directionOutput, 0, len(r.Directions))
	for _, d := range r.Directions {
		directions = append(directions, directionOutput{Label: d.Label})
	}

	tools := make([]toolOutput, 0, len(r.RequiredTools))
	for _, t := range r.RequiredTools {
		tools = append(tools, toolOutput{ID: t.ID, Name: t.Name})
	}

	return recipeOutput{
		ID:                     r.ID,
		Title:                  r.Title,
		Description:            r.Description,
		RecipeFamily:           r.RecipeFamily,
		Servings:               r.UserCoversCount,
		StaticServings:         r.StaticCoversCount,
		PreparationTime:        r.PreparationTime,
		CookingTime:            r.CookingTime,
		RestingTime:            r.RestingTime,
		ImageURL:               r.ImageURL,
		Tip:                    r.Tip.Description,
		Constituents:           toConstituentOutputs(r.Constituents),
		AdditionalConstituents: toConstituentOutputs(r.AdditionalConstituents),
		Directions:             directions,
		Tools:                  tools,
	}
}
