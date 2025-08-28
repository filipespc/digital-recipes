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
    1.  **AI Extraction & Linking:** The AI extracts ingredient strings and automatically links them to a canonical ingredient in the master list (e.g., "2 large eggs" -> `Canonical: Egg`).
    2.  **AI Suggestion:** If an ingredient is unrecognized (e.g., "za'atar"), the AI flags it as a `Suggested New Ingredient`.
    3.  **User Review UI:** The user is presented with a list of ingredients in different states: `Confirmed Link`, `Suggested New`, and allows for `User Override`.
    * The override mechanism must allow the user to:
        * Re-link to a different existing canonical ingredient.
        * Create a new canonical ingredient from the current string (e.g., force "bread flour" to be a new ingredient instead of linking to "Flour").
* **Acceptance Criteria:**
    * `GIVEN` I am on the "Review & Edit" screen.
    * `THEN` I see fields for `Title`, `Servings`, `Ingredients`, `Instructions`, and `Tips & Observations` populated by the AI **and presented in editable input controls**.
    * `WHEN` I modify the text in the `Title`, `Servings`, `Instructions`, or `Tips & Observations` fields.
    * `THEN` my changes are reflected in the input fields, ready to be saved.
    * `AND` the system automatically links most ingredients to their canonical form.
    * `AND` ingredients the AI identifies as new are clearly highlighted as suggestions.
    * `WHEN` I see a suggested new ingredient, I can confirm its creation with a single click.
    * `WHEN` I spot an incorrect ingredient link, I can easily click on it and choose to create a new canonical ingredient or re-link to an existing one from an override menu.
    * `WHEN` I am satisfied with all my edits and tap "Save".
    * `THEN` all changes to all fields are saved to the database.

### 4.3. View Recipe List & Details

* **User Story:** "As a cook, I want to see all my saved recipes in a clean list, and when I'm ready to cook, I want to view a single recipe in a clear, easy-to-read format."
* **Acceptance Criteria:**
    * `GIVEN` I am on the main screen of the app.
    * `THEN` I see a scrollable list of all my saved recipe titles.
    * `WHEN` I tap on a recipe title.
    * `THEN` I am navigated to the Recipe Detail screen.
    * `AND` the Detail screen cleanly displays the `Title`, `Servings`, `Ingredients`, `Instructions`, and `Tips & Observations`.