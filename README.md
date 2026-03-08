# jow-claude-plugin

A Claude Code MCP plugin to manage recipes on [jow.fr](https://jow.fr) directly from Claude.

## Setup

### 1. Build the server

```bash
go build -o build/jow-mcp-server ./cmd/main
```

### 2. Get a Jow token

Log in to jow.fr, open DevTools → Network, and copy the `Authorization: Bearer <token>` value from any API request.

```bash
export JOW_TOKEN=<your-token>
```

### 3. Connect Claude Code

Install the plugin via the Claude Code plugin system. The `.mcp.json` at the repo root configures Claude Code to launch the binary automatically using stdio transport.

The binary also supports HTTP when `JOW_MCP_HTTP_ADDR` is set:

```bash
JOW_TOKEN=<token> JOW_MCP_HTTP_ADDR=127.0.0.1:3000 ./build/jow-mcp-server
```

## Tools

- **`search_ingredients`** — Search Jow's ingredient catalog by name (French preferred). Returns ingredient IDs and all available unit IDs.
- **`get_recipe_tools`** — List available kitchen tools (wok, casserole, four, …) with their IDs.
- **`create_recipe`** — Create a full recipe on Jow: title, servings, ingredients with quantities and units, ordered directions, kitchen tools, and an optional tip.

## Example prompt

> "Create the udon boulettes de boeuf recipe for 4 servings: 400g udon noodles, 300g ground beef, 2 tbsp soy sauce, 1 tbsp sesame oil, 1 pinch of salt. Steps: …"
