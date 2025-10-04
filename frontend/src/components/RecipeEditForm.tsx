'use client';

import { useState, useEffect, useCallback, useRef } from 'react';
import { useRouter } from 'next/navigation';
import { Recipe, RecipeWithIngredients, RecipeIngredient } from '@/types/recipe';
import { RecipeAPI } from '@/services/api';
import IngredientList from './IngredientList';

interface RecipeEditFormProps {
  recipe: RecipeWithIngredients;
}

export default function RecipeEditForm({ recipe: initialRecipe }: RecipeEditFormProps) {
  const router = useRouter();
  const [recipe, setRecipe] = useState<RecipeWithIngredients>(initialRecipe);
  const [ingredients, setIngredients] = useState<RecipeIngredient[]>(initialRecipe.ingredients || []);
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Auto-save timeout management using useRef to prevent memory leaks
  const saveTimeoutRef = useRef<NodeJS.Timeout | null>(null);

  // Request cancellation to prevent race conditions
  const saveControllerRef = useRef<AbortController | null>(null);
  const saveSequenceRef = useRef(0);

  // Update ingredients when recipe changes (e.g., from refetch)
  useEffect(() => {
    setIngredients(initialRecipe.ingredients || []);
  }, [initialRecipe.ingredients]);

  const handleIngredientsChange = useCallback((newIngredients: RecipeIngredient[]) => {
    setIngredients(newIngredients);
    // Update the recipe state to include the new ingredients
    setRecipe(prev => ({ ...prev, ingredients: newIngredients }));
  }, []);

  const validateField = (name: string, value: string): string | null => {
    // Basic sanitization
    const trimmed = value.trim();

    // Security checks - detect potential script injection attempts
    const dangerousPatterns = [
      /<script[^>]*>/i,
      /javascript:/i,
      /data:text\/html/i,
      /on\w+\s*=/i, // Event handlers like onclick, onload
      /<iframe[^>]*>/i,
      /<object[^>]*>/i,
      /<embed[^>]*>/i
    ];

    for (const pattern of dangerousPatterns) {
      if (pattern.test(value)) {
        return 'Invalid content detected. HTML/JavaScript is not allowed.';
      }
    }

    switch (name) {
      case 'title':
        if (!trimmed) return 'Title is required';
        if (trimmed.length < 3) return 'Title must be at least 3 characters';
        if (trimmed.length > 200) return 'Title must be less than 200 characters';

        // Check for excessive special characters that might indicate spam
        const specialCharRatio = (trimmed.match(/[!@#$%^&*()_+={}\[\]|\\:";'<>?,./]/g) || []).length / trimmed.length;
        if (specialCharRatio > 0.3) return 'Title contains too many special characters';

        return null;

      case 'servings':
        if (!value) return null; // Optional field
        if (trimmed.length > 50) return 'Servings must be less than 50 characters';

        // Basic format validation for servings
        if (trimmed && !/^[\w\s\-,0-9]+$/.test(trimmed)) {
          return 'Servings should only contain letters, numbers, spaces, hyphens, and commas';
        }

        return null;

      case 'instructions':
        if (!value) return null; // Optional field
        if (trimmed.length > 10000) return 'Instructions must be less than 10,000 characters';

        // Check for reasonable content structure
        if (trimmed.length > 0) {
          const lines = trimmed.split('\n').filter(line => line.trim());
          if (lines.length > 100) return 'Too many instruction steps (max 100 lines)';

          // Check for excessively long lines that might be spam
          const longLines = lines.filter(line => line.length > 500);
          if (longLines.length > 0) return 'Individual instruction steps should be shorter than 500 characters';
        }

        return null;

      case 'tips':
        if (!value) return null; // Optional field
        if (trimmed.length > 2000) return 'Tips must be less than 2,000 characters';

        // Check for reasonable content structure
        if (trimmed.length > 0) {
          const lines = trimmed.split('\n').filter(line => line.trim());
          if (lines.length > 20) return 'Too many tip entries (max 20 lines)';
        }

        return null;

      default:
        return null;
    }
  };

  const handleFieldChange = useCallback((field: keyof Recipe, value: string) => {
    // Validate the field
    const error = validateField(field, value);
    setErrors(prev => ({
      ...prev,
      [field]: error || ''
    }));

    // Update the recipe state
    setRecipe(prev => ({
      ...prev,
      [field]: value
    }));

    // Clear any existing save timeout
    if (saveTimeoutRef.current) {
      clearTimeout(saveTimeoutRef.current);
      saveTimeoutRef.current = null;
    }

    // Only auto-save if there are no validation errors
    if (!error) {
      setSaveStatus('saving');
      saveTimeoutRef.current = setTimeout(async () => {
        try {
          await saveRecipe({ [field]: value });
          setSaveStatus('saved');
          setTimeout(() => setSaveStatus('idle'), 2000);
        } catch (err) {
          console.error('Auto-save failed:', err);
          setSaveStatus('error');
          setTimeout(() => setSaveStatus('idle'), 3000);
        }
      }, 1000); // 1 second debounce
    }
  }, []); // No dependencies needed since we use useRef

  const saveRecipe = async (updates: Partial<Recipe>) => {
    // Cancel previous request if it exists
    if (saveControllerRef.current) {
      saveControllerRef.current.abort();
    }

    // Create new abort controller for this request
    const controller = new AbortController();
    saveControllerRef.current = controller;

    // Increment sequence number to track request order
    const currentSequence = ++saveSequenceRef.current;

    setSaving(true);
    try {
      // Merge updates with current recipe data to ensure all required fields are sent
      const fullUpdate = {
        title: recipe.title, // Always include title (required)
        servings: recipe.servings,
        instructions: recipe.instructions,
        tips: recipe.tips,
        ...updates // Override with new values
      };

      const updatedRecipe = await RecipeAPI.updateRecipe(recipe.id, fullUpdate);

      // Only update state if this is the latest request (not cancelled)
      if (currentSequence === saveSequenceRef.current && !controller.signal.aborted) {
        setRecipe(prev => ({ ...prev, ...updatedRecipe }));
      }
    } catch (error) {
      // Don't log or throw errors for cancelled requests
      if (error instanceof Error && error.name === 'AbortError') {
        return; // Request was cancelled, this is expected
      }

      // Only handle real errors if this is still the latest request
      if (currentSequence === saveSequenceRef.current) {
        console.error('Failed to save recipe:', error);
        throw error;
      }
    } finally {
      // Only update loading state if this is still the latest request
      if (currentSequence === saveSequenceRef.current) {
        setSaving(false);
        saveControllerRef.current = null;
      }
    }
  };

  // Check if all ingredients are resolved (linked to pantry items)
  const getUnresolvedIngredients = useCallback(() => {
    return ingredients.filter(ingredient => !ingredient.canonical_ingredient_id && !ingredient.pantry_item_id);
  }, [ingredients]);

  const unresolvedIngredients = getUnresolvedIngredients();
  const hasUnresolvedIngredients = unresolvedIngredients.length > 0;

  const handlePublish = async () => {
    // Validate all required fields
    const titleError = validateField('title', recipe.title);
    if (titleError) {
      setErrors(prev => ({ ...prev, title: titleError }));
      return;
    }

    try {
      setSaving(true);

      // Auto-create pantry items for unresolved ingredients
      if (hasUnresolvedIngredients && ingredients) {
        const creationPromises = [];

        for (const ingredient of ingredients) {
          if (!ingredient.pantry_item_id) {
            // Get AI suggestion for pantry item name
            const suggestionPromise = RecipeAPI.suggestPantryItemName(ingredient.original_text)
              .then(suggestion => ({
                ingredient,
                suggestedName: suggestion.suggested_name
              }))
              .catch(() => ({
                ingredient,
                suggestedName: ingredient.original_text // Fallback to original text
              }));

            creationPromises.push(suggestionPromise);
          }
        }

        // Process all suggestions and create pantry items
        const suggestionsToCreate = await Promise.all(creationPromises);

        for (const { ingredient, suggestedName } of suggestionsToCreate) {
          try {
            // Create the pantry item
            const pantryItem = await RecipeAPI.createPantryItem(suggestedName, 1); // Using userId 1 for now

            // Link it to the ingredient
            await RecipeAPI.linkIngredientToPantryItem(
              recipe.id,
              ingredient.id,
              pantryItem.id
            );

            // Update local state
            setIngredients(prev => prev?.map(ing =>
              ing.id === ingredient.id
                ? { ...ing, pantry_item_id: pantryItem.id, pantry_item_name: pantryItem.name }
                : ing
            ) || []);
          } catch (err) {
            console.error(`Failed to create pantry item for "${suggestedName}":`, err);
          }
        }
      }

      // Now publish the recipe
      await RecipeAPI.updateRecipeStatus(recipe.id, 'published');
      setRecipe(prev => ({ ...prev, status: 'published' }));
      setSaveStatus('saved');

      // Clear any ingredient errors on successful publish
      setErrors(prev => {
        const { ingredients, ...rest } = prev;
        return rest;
      });

      // Redirect to recipe view page after successful publish
      setTimeout(() => {
        router.push(`/recipes/${recipe.id}`);
      }, 500); // Small delay to show success state
    } catch (error) {
      console.error('Failed to publish recipe:', error);
      setSaveStatus('error');
      setTimeout(() => setSaveStatus('idle'), 3000);
    } finally {
      setSaving(false);
    }
  };

  // Cleanup timeout and cancel pending requests on unmount
  useEffect(() => {
    return () => {
      if (saveTimeoutRef.current) {
        clearTimeout(saveTimeoutRef.current);
        saveTimeoutRef.current = null;
      }
      if (saveControllerRef.current) {
        saveControllerRef.current.abort();
        saveControllerRef.current = null;
      }
    };
  }, []); // Empty dependency array since we use useRef

  const getSaveStatusDisplay = () => {
    switch (saveStatus) {
      case 'saving':
        return (
          <div className="flex items-center text-blue-600 text-sm">
            <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600 mr-2"></div>
            Saving...
          </div>
        );
      case 'saved':
        return (
          <div className="flex items-center text-green-600 text-sm">
            <svg className="h-4 w-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M5 13l4 4L19 7" />
            </svg>
            Saved
          </div>
        );
      case 'error':
        return (
          <div className="flex items-center text-red-600 text-sm">
            <svg className="h-4 w-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.5 0L5.268 16.5c-.77.833.192 2.5 1.732 2.5z" />
            </svg>
            Save failed
          </div>
        );
      default:
        return null;
    }
  };

  return (
    <form className="space-y-6">
      {/* Save Status Indicator */}
      <div className="flex justify-between items-center">
        <div className="text-sm text-gray-600">
          Last updated: {new Date(recipe.updated_at).toLocaleDateString('en-US', {
            month: 'short',
            day: 'numeric',
            year: 'numeric',
            hour: 'numeric',
            minute: '2-digit',
          })}
        </div>
        {getSaveStatusDisplay()}
      </div>

      {/* Title Field */}
      <div>
        <label htmlFor="title" className="block text-sm font-medium text-gray-700">
          Recipe Title *
        </label>
        <input
          type="text"
          id="title"
          value={recipe.title}
          onChange={(e) => handleFieldChange('title', e.target.value)}
          className={`mt-1 block w-full px-3 py-2 border ${
            errors.title ? 'border-red-300' : 'border-gray-300'
          } rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm`}
          placeholder="Enter recipe title..."
        />
        {errors.title && (
          <p className="mt-1 text-sm text-red-600">{errors.title}</p>
        )}
      </div>

      {/* Servings Field */}
      <div>
        <label htmlFor="servings" className="block text-sm font-medium text-gray-700">
          Servings
        </label>
        <input
          type="text"
          id="servings"
          value={recipe.servings || ''}
          onChange={(e) => handleFieldChange('servings', e.target.value)}
          className={`mt-1 block w-full px-3 py-2 border ${
            errors.servings ? 'border-red-300' : 'border-gray-300'
          } rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm`}
          placeholder="e.g., 4 servings, 6-8 portions..."
        />
        {errors.servings && (
          <p className="mt-1 text-sm text-red-600">{errors.servings}</p>
        )}
      </div>

      {/* Instructions Field */}
      <div>
        <label htmlFor="instructions" className="block text-sm font-medium text-gray-700">
          Instructions
        </label>
        <textarea
          id="instructions"
          rows={12}
          value={recipe.instructions || ''}
          onChange={(e) => handleFieldChange('instructions', e.target.value)}
          className={`mt-1 block w-full px-3 py-2 border ${
            errors.instructions ? 'border-red-300' : 'border-gray-300'
          } rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm`}
          placeholder="Enter step-by-step instructions..."
        />
        <p className="mt-1 text-xs text-gray-500">
          {recipe.instructions ? recipe.instructions.length : 0} / 10,000 characters
        </p>
        {errors.instructions && (
          <p className="mt-1 text-sm text-red-600">{errors.instructions}</p>
        )}
      </div>

      {/* Tips Field */}
      <div>
        <label htmlFor="tips" className="block text-sm font-medium text-gray-700">
          Tips & Observations
        </label>
        <textarea
          id="tips"
          rows={6}
          value={recipe.tips || ''}
          onChange={(e) => handleFieldChange('tips', e.target.value)}
          className={`mt-1 block w-full px-3 py-2 border ${
            errors.tips ? 'border-red-300' : 'border-gray-300'
          } rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm`}
          placeholder="Add any tips, notes, or observations about this recipe..."
        />
        <p className="mt-1 text-xs text-gray-500">
          {recipe.tips ? recipe.tips.length : 0} / 2,000 characters
        </p>
        {errors.tips && (
          <p className="mt-1 text-sm text-red-600">{errors.tips}</p>
        )}
      </div>

      {/* Ingredients Management */}
      <div className="border-t border-gray-200 pt-6">
        <IngredientList
          recipeId={recipe.id}
          ingredients={ingredients}
          onIngredientsChange={handleIngredientsChange}
        />
      </div>

      {/* Action Buttons */}
      <div className="flex items-center justify-between pt-6 border-t border-gray-200">
        <div className="text-sm text-gray-600">
          {recipe.status === 'review_required' && (
            <span className="text-yellow-600">
              ⚠️ This recipe is ready for review and publishing
            </span>
          )}
          {recipe.status === 'published' && (
            <span className="text-green-600">
              ✅ This recipe is published
            </span>
          )}
        </div>

        <div className="flex flex-col space-y-3">
          {/* Ingredient resolution status */}
          {recipe.status === 'review_required' && (
            <div className="text-sm">
              {hasUnresolvedIngredients ? (
                <div className="flex items-center space-x-2 text-blue-700 bg-blue-50 px-3 py-2 rounded-md">
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span>
                    {unresolvedIngredients.length} new pantry item(s) will be created when you publish
                  </span>
                </div>
              ) : ingredients.length > 0 ? (
                <div className="flex items-center space-x-2 text-green-700 bg-green-50 px-3 py-2 rounded-md">
                  <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                  </svg>
                  <span>All ingredients resolved - ready to publish!</span>
                </div>
              ) : null}
            </div>
          )}

          {/* Error message for ingredients */}
          {errors.ingredients && (
            <div className="text-sm text-red-600 bg-red-50 px-3 py-2 rounded-md">
              {errors.ingredients}
            </div>
          )}

          <div className="flex space-x-4">
            {recipe.status === 'review_required' && (
              <button
                type="button"
                onClick={handlePublish}
                disabled={saving || !!errors.title}
                className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {saving ? 'Publishing...' : hasUnresolvedIngredients ? `Publish & Create ${unresolvedIngredients.length} Pantry Items` : 'Publish Recipe'}
              </button>
            )}
          </div>
        </div>
      </div>
    </form>
  );
}