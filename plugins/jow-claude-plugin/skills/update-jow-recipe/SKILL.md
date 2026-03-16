---
name: update-jow-recipe
description: >
  Met à jour une recette existante sur jow.fr. Déclencher dès que l'utilisateur veut modifier,
  corriger ou mettre à jour une recette déjà publiée sur Jow — qu'il donne un titre, un ID, une
  URL ou qu'il dise simplement "modifie la recette X", "corrige les ingrédients", "change les
  étapes", "met à jour la recette". Utilise le tool update_recipe ainsi que tous les outils de
  résolution d'ingrédients et d'ustensiles déjà disponibles dans create-jow-recipe.
---

# Mettre à jour une recette Jow

Ce skill suit le même pipeline que `create-jow-recipe` pour la résolution des ingrédients,
ustensiles et métadonnées. Consulte ce skill pour toutes les règles détaillées (sélection
d'unités, règles de servings, règles de description, quantités dans les étapes, etc.). La
seule différence majeure est que tu travailles sur une recette **existante** et que tu appelles
`update_recipe` (PUT) à la place de `create_recipe` (POST).

## Step 1 — Identifier la recette à modifier

L'utilisateur peut désigner la recette de plusieurs façons :

- **ID Jow** (ex. `69b1df130e1d3d40a7a049a3`) → utilise-le directement
- **URL Jow** (ex. `https://jow.fr/user-recipes/69b1df130e1d3d40a7a049a3`) → extrais l'ID de l'URL
- **Titre** → appelle `get_recipes` pour lister les recettes de l'utilisateur et trouve la
  recette dont le titre correspond. Si plusieurs recettes ont un titre similaire, liste-les et
  demande à l'utilisateur de confirmer laquelle modifier.

Si l'utilisateur ne précise pas du tout de recette, demande-lui : "Quelle recette souhaitez-vous
modifier ? Donnez-moi son titre, son ID ou son URL Jow."

## Step 2 — Comprendre les modifications souhaitées

L'utilisateur peut fournir :
- Un **nouveau document source** (PDF, image, URL, texte) contenant la version corrigée → traite-le
  comme dans le Step 1 de `create-jow-recipe` pour extraire le contenu complet
- Des **modifications ponctuelles** exprimées en langage naturel (ex. "ajoute du piment", "change
  le temps de cuisson à 20 min", "remplace le beurre par de l'huile d'olive") → applique uniquement
  ces changements sur la recette existante

Si les modifications sont ponctuelles et que tu as besoin du contenu actuel de la recette pour
appliquer les changements correctement, demande à l'utilisateur de le fournir ou de confirmer
les valeurs actuelles.

## Step 3 — Résoudre les ingrédients

Applique exactement les mêmes règles que dans le **Step 3 de `create-jow-recipe`** :
- `search_ingredients` pour chaque ingrédient (nouveaux ou modifiés)
- Sélection de l'unité correspondant à ce que dit la recette source
- `quantity_per_cover` = quantité totale ÷ nombre de portions
- Signalement des substitutions à l'utilisateur

Pour les ingrédients **inchangés**, réutilise les IDs et unit_ids déjà connus si l'utilisateur
les fournit. Sinon, résous-les de nouveau via `search_ingredients`.

## Step 4 — Résoudre les ustensiles

Même règles que le **Step 4 de `create-jow-recipe`** : appelle `get_recipe_tools` une seule fois
et mappe chaque ustensile.

## Step 5 — Extraire les étapes

Même règles que le **Step 5 de `create-jow-recipe`** :
- Respecte l'organisation des étapes de la source
- Précise les quantités à chaque étape pour les ingrédients partagés

## Step 6 — Déterminer les métadonnées

Mêmes règles que le **Step 6 de `create-jow-recipe`** pour `title`, `description`,
`recipe_family`, `servings`, `static_servings`, temps de préparation/cuisson/repos, `tip`.

Pour une modification ponctuelle, ne change que le champ concerné et conserve les autres à
l'identique.

## Step 7 — Présenter un résumé et confirmer

Présente la recette complète dans le même format que le **Step 7 de `create-jow-recipe`**, en
mettant en évidence les **modifications** par rapport à la version précédente :

---

**🍽 [Titre]**
*[Description]*
*Famille : X · Portions : N · Préparation : Xmin · Cuisson : Xmin*

**Ingrédients** *(⚠ = modifié, + = ajouté, − = supprimé)*
- X [unité] de [ingrédient] *(id: …, unit_id: …)*
- …

**Ustensiles**
- [outil] *(id: …)*

**Étapes**
1. …
2. …

**Conseil** *(si disponible)*
> …

---

Demande confirmation : "Est-ce que tout vous semble correct ? Je vais mettre à jour la recette
**[titre]** (ID : `[id]`) sur Jow."

## Step 8 — Uploader l'image (si nécessaire)

Si l'utilisateur fournit une nouvelle image :
1. Demande le chemin du fichier si ce n'est pas déjà fourni
2. Appelle `upload_recipe_image` et note le `image_url` retourné

Si l'utilisateur ne fournit pas de nouvelle image, conserve l'`image_url` existante telle quelle.

## Step 9 — Mettre à jour la recette

Une fois confirmé :

1. Appelle `update_recipe` avec **tous les champs** (y compris ceux inchangés) et l'`id` de la
   recette. `update_recipe` remplace la recette complète — un champ omis sera perdu.
2. Partage l'URL de la recette mise à jour avec l'utilisateur.
3. Si le `recipe_family` a changé, propose d'ajouter la recette à la collection correspondante
   (même logique que le Step 9 de `create-jow-recipe`) en demandant d'abord confirmation.
   Si confirmé : appelle d'abord `favorite_recipe`, puis `add_recipe_to_collection`.
