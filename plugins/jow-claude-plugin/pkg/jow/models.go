package jow

import "time"

// Unit represents a measurement unit (kg, g, ml, piece, etc.)
// The Jow API uses "id" in search results but "_id" in ingredient detail responses.
type Unit struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AlternativeUnit pairs a unit with a reference quantity
type AlternativeUnit struct {
	Unit     Unit    `json:"unit"`
	Quantity float64 `json:"quantity"`
}

// DisplayableUnit is like an AlternativeUnit when showed in the UI
type DisplayableUnit struct {
	Unit Unit `json:"unit"`
}

// Ingredient represents a Jow ingredient.
type Ingredient struct {
	ID                 string             `json:"id"`
	Name               string             `json:"name"`
	ImageURL           string             `json:"imageUrl"`
	IsBasicIngredient  bool               `json:"isBasicIngredient"`
	RecipeUploadConfig RecipeUploadConfig `json:"recipeUploadConfig"`
	NaturalUnit        Unit               `json:"naturalUnit"`
	AlternativeUnits   []AlternativeUnit  `json:"alternativeUnits"`
	DisplayableUnits   []DisplayableUnit  `json:"displayableUnits"`
}

type RecipeUploadConfig struct {
	IsAdditionalConstituent bool `json:"isAdditionalConstituent"`
}

// Constituent is an ingredient with its quantity in a recipe
type Constituent struct {
	Ingredient       Ingredient `json:"ingredient"`
	QuantityPerCover float64    `json:"quantityPerCover"`
	Unit             Unit       `json:"unit"`
}

// Direction is a recipe step
type Direction struct {
	Label               string       `json:"label"`
	InvolvedIngredients []Ingredient `json:"involvedIngredients"`
}

// Tool is a kitchen utensil
type Tool struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ImageURL string `json:"imageUrl"`
}

// Recipe is the body sent to create or update a recipe
type Recipe struct {
	ID                     string            `json:"id"`
	AdditionalConstituents []Constituent     `json:"additionalConstituents"`
	BackgroundPattern      BackgroundPattern `json:"backgroundPattern"`
	Constituents           []Constituent     `json:"constituents"`
	CookingTime            int               `json:"cookingTime"`
	CreatedAt              *time.Time        `json:"createdAt"`
	Directions             []Direction       `json:"directions"`
	ImageURL               string            `json:"imageUrl,omitempty"`
	PlaceHolderURL         string            `json:"placeHolderUrl"`
	PreparationTime        int               `json:"preparationTime"`
	RecipeFamily           string            `json:"recipeFamily,omitempty"`
	RestingTime            *int              `json:"restingTime"`
	RequiredTools          []Tool            `json:"requiredTools"`
	StaticCoversCount      bool              `json:"staticCoversCount"`
	Tip                    Tip               `json:"tip"`
	Title                  string            `json:"title"`
	UserConstituents       []Constituent     `json:"userConstituents"`
	UserCoversCount        int               `json:"userCoversCount"`
}

type BackgroundPattern struct {
	Color    string `json:"color"`
	ImageUrl string `json:"imageUrl"`
}

type Tip struct {
	Description string `json:"description"`
}

// apiResponse wraps the standard Jow API envelope
type apiResponse[T any] struct {
	Meta interface{} `json:"meta"`
	Data T           `json:"data"`
}
