package jow

import (
	"bytes"
	"cmp"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"path/filepath"
	"slices"
)

// GetMostRecentRecipeByTitle returns the most recently created recipe matching the title in the user's recipes
func (c *Client) GetMostRecentRecipeByTitle(title string) (*Recipe, error) {
	recipes, err := c.GetRecipes()
	if err != nil {
		return nil, fmt.Errorf("failed to get recipes: %v", err)
	}

	slices.SortStableFunc(recipes, func(r1, r2 Recipe) int {
		return cmp.Compare(r1.CreatedAt.UnixMicro(), r2.CreatedAt.UnixMicro())
	})

	for _, r := range slices.Backward(recipes) {
		if r.Title == title {
			return &r, nil
		}
	}

	return nil, fmt.Errorf("recipe %s not found", title)
}

// GetRecipes returns all user-created recipes
func (c *Client) GetRecipes() ([]Recipe, error) {
	path := fmt.Sprintf("/recipes/uploaded")

	body, err := c.do("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("get recipes: %w", err)
	}

	type recipesResponse struct {
		Recipes []Recipe `json:"recipes"`
	}

	var result apiResponse[recipesResponse]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse get recipes response: %w", err)
	}
	return result.Data.Recipes, nil
}

// SearchIngredients searches Jow's ingredient catalog by name.
// Returns up to limit results sorted by relevance.
func (c *Client) SearchIngredients(query string, limit int) ([]Ingredient, error) {
	path := fmt.Sprintf("/ingredients/search?query=%s&limit=%d&start=0&availabilityZoneId=%s",
		url.QueryEscape(query), limit, availabilityZone)

	body, err := c.do("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("search ingredients %q: %w", query, err)
	}

	// The search endpoint returns either a bare array or a {meta,data} envelope.
	if len(body) > 0 && body[0] == '[' {
		var ingredients []Ingredient
		if err := json.Unmarshal(body, &ingredients); err != nil {
			return nil, fmt.Errorf("parse search response: %w", err)
		}
		return ingredients, nil
	}
	var result apiResponse[[]Ingredient]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}
	return result.Data, nil
}

// GetIngredient fetches the full ingredient details by ID.
func (c *Client) GetIngredient(id string) (*Ingredient, error) {
	path := fmt.Sprintf("/ingredient/%s?availabilityZoneId=%s&withRecipes=false", id, availabilityZone)

	body, err := c.do("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("get ingredient %s: %w", id, err)
	}

	var result apiResponse[Ingredient]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse ingredient response: %w", err)
	}
	return &result.Data, nil
}

// GetRecipeTools fetches the list of available kitchen tools.
func (c *Client) GetRecipeTools() ([]Tool, error) {
	path := fmt.Sprintf("/recipetools?availabilityZoneId=%s", availabilityZone)

	body, err := c.do("GET", path, nil)
	if err != nil {
		return nil, fmt.Errorf("get recipe tools: %w", err)
	}

	var result apiResponse[[]Tool]
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse tools response: %w", err)
	}
	return result.Data, nil
}

// CreateRecipe creates a new uploaded recipe with the full payload and returns its ID.
func (c *Client) CreateRecipe(req Recipe) error {
	_, err := c.do("POST", "/recipes/uploaded", req)
	if err != nil {
		return fmt.Errorf("create recipe: %w", err)
	}
	return nil
}

// UploadRecipeImage uploads an image file to Jow and returns the imageUrl (e.g. "uploadedrecipes/xyz.jpg").
func (c *Client) UploadRecipeImage(imagePath string) (string, error) {
	f, err := os.Open(imagePath)
	if err != nil {
		return "", fmt.Errorf("open image: %w", err)
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)

	mimeType := detectMimeType(f)
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image"; filename="%s"`, filepath.Base(imagePath)))
	h.Set("Content-Type", mimeType)

	part, err := w.CreatePart(h)
	if err != nil {
		return "", fmt.Errorf("create form part: %w", err)
	}
	if _, err = io.Copy(part, f); err != nil {
		return "", fmt.Errorf("copy image data: %w", err)
	}
	err = w.Close()
	if err != nil {
		return "", fmt.Errorf("close form part: %w", err)
	}

	req, err := http.NewRequest("POST", baseURL+"/recipes/uploaded/image", &buf)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Accept-Language", "fr")
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Origin", "https://jow.fr")
	req.Header.Set("Referer", "https://jow.fr/")
	req.Header.Set("x-jow-withmeta", "1")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result apiResponse[struct {
		ImageURL string `json:"imageUrl"`
	}]
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("parse upload response: %w", err)
	}
	return result.Data.ImageURL, nil
}

func detectMimeType(file *os.File) string {
	mimeType := mime.TypeByExtension(filepath.Ext(file.Name()))
	if mimeType != "" {
		return mimeType
	}

	buf := make([]byte, 512)
	n, _ := file.Read(buf)
	file.Seek(0, io.SeekStart)
	return http.DetectContentType(buf[:n])
}

// UpdateRecipe updates an existing uploaded recipe.
func (c *Client) UpdateRecipe(id string, req Recipe) error {
	_, err := c.do("PUT", "/recipes/uploaded/"+id, req)
	if err != nil {
		return fmt.Errorf("update recipe %s: %w", id, err)
	}
	return nil
}
