# PRD: AI-Powered Recipe Hub MVP

**Document Status:** Final v1.0
**Author:** AIPM
**Last Updated:** August 28, 2025

## 1. Introduction & Problem Statement

Cooking enthusiasts and busy individuals collect recipes from various sources (screenshots, photos of physical books, etc.), leading to a scattered and disorganized library. Existing tools like Trello or simple note-taking apps are not designed for recipe management, making the process of adding, finding, and deciding what to cook a significant point of friction.

This document outlines the Minimum Viable Product (MVP) for an AI-Powered Recipe Hub designed to solve these core problems by automating data entry and providing intelligent search capabilities.

## 2. User Persona & Job-to-be-Done

* **Persona:** The Busy Planner
* **Problem:** "I want to plan my meals, but I always struggle to remember all my recipe options. I've tried using generic tools like Trello, but the process is painful: input is tedious and manual, browsing for ideas is inefficient, and the format isn't suited for cooking."
* **Job-to-be-Done (JTBD):** "Help me consolidate my recipes into one structured, searchable place so I can quickly and easily decide what to cook."

## 3. MVP Scope & Guiding Principles

### Guiding Principles

* **Automate by Default, Allow Correction by Exception:** The AI should handle the heavy lifting of data entry and organization. The user's primary role is to review and make minor corrections, not to perform manual labor.
* **Structured Data is Foundational:** Every recipe will be stored in a structured format from day one. This is non-negotiable as it powers the core search and retrieval functionality.
* **Focus on the Core Loop:** The MVP is exclusively focused on the `Save -> Find -> Decide` loop. All other functionality is out of scope.

### In Scope for MVP (Must-Haves)

1.  **Add Recipe from Image(s):** AI-powered extraction from one or more images.
2.  **Review & Edit Recipe:** A crucial workflow to review, correct, and confirm the AI's output, especially for structured ingredients.
3.  **View Recipe List:** A central library of all saved recipes.
4.  **View Recipe Details:** A clean, consumption-focused view for cooking.

### Out of Scope for MVP (Parking Lot)

The following features are valuable but will be considered for future versions:
* Smart Search through natural language
* Add recipes from a URL
* Manual recipe creation form
* Recipe tagging
* Shopping list generation
* Ingredient usage prediction
* AI-powered shopping cart scanning
* AI-powered recipe translation from multiple languages

## 4. Feature Specifications

### 4.1. Add Recipe from Image(s)

* **User Story:** "As a busy cook, I want to upload one or more photos of a recipe so that the app can automatically analyze them, combine the information, and structure the content for me."
* **Acceptance Criteria:**
    * `GIVEN` I am on the main recipe list screen.
    * `WHEN` I tap the "Add Recipe from Image" button.
    * `THEN` I am prompted to select one or more images from my device's gallery.
    * `GIVEN` I have selected my image(s).
    * `WHEN` I confirm the selection.
    * `THEN` the app displays a loading indicator while the AI processes the images as a single cohesive recipe.
    * `AND` upon completion, I am navigated to the "Review & Edit" screen with the combined, extracted data.

### 4.2. Review & Edit Recipe

* **User Story:** "As a cook, after the AI has processed my recipe, I want to review its output and easily correct any mistakes across all fields, so my saved recipe is perfectly organized with minimal effort."
* **Core Workflow (Ingredients):**
    1.  **AI Extraction & Smart Linking:** The AI extracts ingredient strings and automatically attempts to link them to existing ingredients in the user's collection (e.g., "2 large eggs" -> links to existing "Egg" if found).
    2.  **Mandatory Ingredient Resolution:** For each unlinked ingredient, users must make a decision before publishing:
        * **Link to Existing:** Search and select from their existing ingredient collection
        * **Create New:** Confirm creating a new ingredient in their collection
    3.  **Publishing Requirement:** Recipes cannot be published until ALL ingredients are resolved (linked to canonical ingredients)
* **User Interaction Model:**
    * **Default State:** AI attempts to link ingredients automatically. Successfully linked ingredients show as "Linked to [Ingredient Name]"
    * **Ingredient Resolution Process:**
        * **Linked Ingredients:** User can confirm the link is correct OR change to link to a different ingredient
        * **Unlinked Ingredients:** User must resolve by either linking to existing ingredient OR creating new one
        * **Search & Link:** Users can search through their ingredient collection to find matches
        * **Create New:** Users can create new canonical ingredients when no match exists
    * **Publishing Validation:** Save button is disabled until all ingredients are resolved
* **Acceptance Criteria:**
    * `GIVEN` I am on the "Review & Edit" screen.
    * `THEN` I see fields for `Title`, `Servings`, `Ingredients`, `Instructions`, and `Tips & Observations` populated by the AI **and presented in editable input controls**.
    * `WHEN` I modify the text in the `Title`, `Servings`, `Instructions`, or `Tips & Observations` fields.
    * `THEN` my changes are reflected in the input fields, ready to be saved.
    * `AND` each ingredient shows its current state: either "Linked to [Ingredient Name]" or "Needs Resolution"
    * `WHEN` I see an ingredient marked as "Linked to [Ingredient Name]".
    * `THEN` I can confirm the link is correct OR click to change the link to a different ingredient OR mark it as a new ingredient if the linked ingredient doesn't actually exist in my collection yet
    * `WHEN` I see an ingredient marked as "Needs Resolution".
    * `THEN` I can search to link it to an existing ingredient OR create a new ingredient
    * `WHEN` I choose to link an ingredient.
    * `THEN` I can search through my existing ingredients and select one to link to
    * `WHEN` some ingredients still need resolution.
    * `THEN` the "Publish Recipe" button is disabled with a message indicating unresolved ingredients
    * `WHEN` all ingredients are resolved and I tap "Publish Recipe".
    * `THEN` all changes are saved and the recipe status is set to 'published' with all ingredients properly linked.

### 4.3. View Recipe List & Details

* **User Story:** "As a cook, I want to see all my saved recipes in a clean list, and when I'm ready to cook, I want to view a single recipe in a clear, easy-to-read format."
* **Acceptance Criteria:**
    * `GIVEN` I am on the main screen of the app.
    * `THEN` I see a scrollable list of all my saved recipe titles.
    * `WHEN` I tap on a recipe title.
    * `THEN` I am navigated to the Recipe Detail screen.
    * `AND` the Detail screen cleanly displays the `Title`, `Servings`, `Ingredients`, `Instructions`, and `Tips & Observations`.

### 4.4. Manage Ingredients

* **User Story:** "As a cook who has used the app for a while, I want to manage my ingredient collection to fix mistakes and keep my data organized, so my recipes stay consistent and useful."
* **Core Features:**
    1. **View Ingredient Collection:** See all canonical ingredients in my collection with usage statistics
    2. **Merge Duplicate Ingredients:** Combine ingredients that represent the same thing (e.g., "Flour" + "All-Purpose Flour")
    3. **Delete Unused Ingredients:** Remove ingredients that aren't used in any recipes
    4. **Rename Ingredients:** Fix typos or improve naming consistency
* **Acceptance Criteria:**
    * `GIVEN` I am on the Ingredient Management screen.
    * `THEN` I see a list of all my canonical ingredients with usage counts (e.g., "Eggs - used in 5 recipes").
    * `WHEN` I identify duplicate ingredients.
    * `THEN` I can select two ingredients and merge them, with all recipe links updating to the primary ingredient.
    * `WHEN` I see an ingredient with 0 recipe usage.
    * `THEN` I can delete it from my collection.
    * `WHEN` I want to fix a typo in an ingredient name.
    * `THEN` I can edit the name and it updates across all linked recipes.
    * `AND` all changes maintain data integrity and recipe consistency.