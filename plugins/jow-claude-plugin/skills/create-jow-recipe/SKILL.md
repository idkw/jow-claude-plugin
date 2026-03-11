---
name: create-jow-recipe
description: >
  Creates a complete recipe on jow.fr from a user-provided document (PDF, image, Google Docs URL, text, etc.).
  Invoke this skill whenever the user shares a recipe — as a file, a URL, a photo, or pasted text — and asks
  to add it to Jow, import it, create it, or publish it. Also trigger when the user says things like "ajoute
  cette recette sur Jow", "crée la recette", "importe cette recette", or shows any recipe content alongside
  a request to put it on jow.fr. Don't wait for the user to spell out the full workflow; start the process
  as soon as recipe content and a Jow intent are present.
---

# Create a Jow Recipe from a Document

## Step 1 — Fetch and read the document

The user will provide a recipe source. It can be:
- A local file path (PDF, image, txt, docx, …) → read it directly with available tools
- A URL (Google Docs, webpage, …) → fetch it with available tools
- Pasted text or an image in the conversation → use it as-is

Read the full content before doing anything else. If the source is unclear or you can't access it, ask the user to paste the content or share it differently.

## Step 2 — Ask for the recipe picture

After reading the document, ask:

> "Avez-vous une photo de la recette à uploader sur Jow ? Si oui, donnez-moi le chemin complet vers le fichier image."

If the user provides a path, keep it for Step 7. If not, continue without an image.
If the user provides a HTTP link, download the file to a temporary file and note its full path for the next steps.

## Step 3 — Extract and resolve ingredients

From the document, extract every ingredient with its quantity and unit exactly as written in the source recipe (e.g. "2 cuillères à soupe de sauce soja", "200 g de bœuf haché").

For **each ingredient**, call `search_ingredients` and pick the best match.

If no good match is found, retry with a broader or simpler query — the ingredient name from the recipe may be too specific for Jow's catalog. Strategies to try in order:
- Remove a qualifying word (e.g. "sauce soja sucrée" → "sauce soja")
- Use the generic name (e.g. "pancetta fumée" → "pancetta", or → "lardons")
- Try a common equivalent (e.g. "crème fraîche épaisse" → "crème fraîche")

Only report the ingredient as not found after at least one retry attempt.

When a substitution was made, two things must follow:
1. **Report it to the user** in the confirmation summary (Step 7) with a note like: *"sauce soja" utilisé à la place de "sauce soja sucrée"*.
2. **In the directions**, use the full original name alongside the substituted name the first time it appears, e.g. "Ajoutez la sauce soja (sauce soja sucrée)…"

### Unit selection — this is critical

The unit you pass to `create_recipe` must match what the source recipe says, not what Jow suggests as "natural":

| Source says | Unit to use |
|---|---|
| "2 cuillères à soupe" | the "Cuillère à soupe" unit id from search results |
| "1 pincée" | the "Pincée" unit id |
| "200 g" / "0.2 kg" | the "Kilogramme" unit id, quantity = 0.2 |
| "3 pièces" / "3 œufs" | the piece/unité unit id |

Rules:
- `unit_id` MUST come from `natural_unit.id` or one of `alternative_units[].id` returned by `search_ingredients`. Never invent one.
- `quantity_per_cover` is always computed as: (total quantity) ÷ (number of servings) — e.g. "2 càs" for 4 servings → `0.5`, unit = Cuillère à soupe

If no good ingredient match is found, note it but don't block the workflow.

## Step 4 — Extract and resolve kitchen tools

Extract every kitchen utensil mentioned in the recipe (wok, casserole, four, poêle, …).

Call `get_recipe_tools` **once** to get all available tools, then match each extracted tool to the closest entry. If a tool has no reasonable match, skip it and tell the user: "L'outil *X* n'a pas été trouvé dans le catalogue Jow et sera ignoré."

## Step 5 — Extract directions

Extract the recipe steps exactly as structured in the source document. Each direction is a coherent group of sentences that belong to the same action — preserve the original pacing and organisation.

- Do NOT merge all steps into one paragraph.
- Do NOT over-split steps; keep sentences that go together in the same direction.
- Write directions in French if the source is French.

### Quantities in directions — this is critical

Whenever an ingredient is split across multiple steps (e.g. butter used both in the dough and in the cream, water used both in the dough and in the syrup), **always state the exact quantity used at each step**. Do not just say "le beurre" — say "les 25 g de beurre doux" or "les 190 g de matière grasse de cuisson". This is essential because Jow only shows the total quantity on the ingredient list, so the user has no way to know how to split it without explicit quantities in the steps.

## Step 6 — Determine recipe metadata

From the document, determine:

| Field | How to determine |
|---|---|
| `title` | Use the recipe title as given |
| `description` | See description rules below |
| `recipe_family` | Pick the most relevant: Plat, Dessert, Apéro, Boisson, Entrée, Autre |
| `servings` | See servings rules below |
| `preparation_time_minutes` | From the document or inferred logically |
| `cooking_time_minutes` | From the document or inferred logically |
| `resting_time_minutes` | Only if the recipe mentions a resting/marinating time |
| `tip` | Any chef's tip present in the document (optional) |

### Description rules

Write a short, appetising description (1–3 sentences) that makes the reader want to cook the recipe. Extract it from the document if a good one is present; otherwise generate it from the recipe content.

For **shared / whole-item** recipes (`static_servings = true`), always include the expected yield at the start of the description:

> "24 cookies moelleux au chocolat. Un classique américain..."
> "1 tarte tatin caramélisée. Un dessert généreux..."

For **per-portion** recipes, no yield mention is needed — just make it enticing.

### Servings rules

First determine **what kind of recipe it is**:

- **Per-portion** — each person gets their own individual serving (assiette, verre, bol, ramequin, …). The number of servings equals the number of people.
- **Shared / whole item** — the recipe produces one indivisible thing to share (gâteau, tarte, fournée de cookies, plat à partager, …).

Then set `servings`, `static_servings`, and compute `quantity_per_cover` accordingly:

| Situation | `servings` | `static_servings` | `quantity_per_cover` |
|---|---|---|---|
| Per-portion, count stated in recipe | that number (e.g. `4`) | `false` | total quantity ÷ servings count |
| Shared/whole-item recipe | number of pieces/units the recipe yields (e.g. `24` cookies, `1` tarte) | `true` | total quantity ÷ servings count |
| Count genuinely cannot be determined | `1` | `true` | total quantity as stated in the recipe |

## Step 7 — Present a summary and confirm

Before calling any creation tool, present the full recipe to the user in this format:

---

**🍽 [Titre]**
*[Description]*
*Famille : X · Portions : N · Préparation : Xmin · Cuisson : Xmin*

**Ingrédients**
- X [unité] de [ingrédient] *(id: …, unit_id: …)*
- …

**Ustensiles**
- [outil] *(id: …)*
- …

**Étapes**
1. …
2. …

**Conseil** *(si disponible)*
> …

---

Then ask: "Est-ce que tout vous semble correct ? Y a-t-il des ajustements à faire avant de créer la recette sur Jow ?"

Incorporate any corrections the user requests before proceeding.

## Step 8 — Upload image and create recipe

Once the user confirms:

1. If an image path was provided in Step 2, call `upload_recipe_image` with that path and note the returned `image_url`.
2. Call `create_recipe` with all the resolved data, including `image_url` if available.
3. Share the returned recipe URL with the user.
