# jow-claude-marketplace

Plugin marketplace for jow.fr Claude Code integration.

## Repo structure

```
.claude-plugin/
  marketplace.json          — marketplace manifest (lists available plugins)
plugins/
  jow-claude-plugin/        — MCP server plugin for jow.fr
    .claude-plugin/
      plugin.json           — plugin manifest
    .mcp.json               — MCP server config (stdio by default)
    cmd/main/main.go        — entrypoint: MCP server setup, tool registration, handlers
    pkg/jow/
      client.go             — HTTP client for the Jow API
      api.go                — API methods: SearchIngredients, GetIngredient, GetRecipeTools, CreateRecipe, …
      models.go             — Go types for Jow API structures
    go.mod / go.sum
```

## Build the plugin binary

```bash
cd plugins/jow-claude-plugin
go build -o build/jow-mcp-server ./cmd/main
```

**stdio (default)** — spawned by Claude Code via `.mcp.json`:
```bash
JOW_TOKEN=<token> ./build/jow-mcp-server
```

**HTTP** — set `JOW_MCP_HTTP_ADDR` to enable streamable HTTP transport:
```bash
JOW_TOKEN=<token> JOW_MCP_HTTP_ADDR=127.0.0.1:3000 ./build/jow-mcp-server
```

### Getting a JOW_TOKEN

Log in to jow.fr, open DevTools → Network, copy the `Authorization: Bearer <token>` header from any request. Store it in `.envrc` (git-ignored):

```bash
export JOW_TOKEN=<token>
```

## Available MCP tools

| Tool | Description |
|------|-------------|
| `search_ingredients` | Search Jow's catalog by name; returns ingredient IDs and all available unit IDs |
| `get_recipe_tools` | List kitchen tools (wok, casserole, four, …) with their IDs |
| `create_recipe` | Create a full recipe on Jow from title, ingredients, directions, tools |

### Workflow for create_recipe

1. Call `search_ingredients` for each ingredient → note `id` and the correct `unit_id`
2. Call `get_recipe_tools` for each tool → note `id`
3. Call `create_recipe` with all data

**Unit rule:** always use the unit the source recipe specifies (e.g. "2 càs" → use the Cuillère à soupe unit ID, not Kilogramme).

## Key conventions

- `quantity_per_cover` = total quantity ÷ number of servings, expressed in the chosen unit
- Ingredients with `recipeUploadConfig.isAdditionalConstituent = true` go in `additionalConstituents` (e.g. salt, butter)
- Recipe family IDs are hardcoded in `resolveRecipeFamily()` in `cmd/main/main.go`
