'use client';

import { useState, useEffect, useCallback } from 'react';
import { Recipe, RecipeWithIngredients } from '@/types/recipe';
import { RecipeAPI } from '@/services/api';

interface RecipeEditFormProps {
  recipe: RecipeWithIngredients;
}

export default function RecipeEditForm({ recipe: initialRecipe }: RecipeEditFormProps) {
  const [recipe, setRecipe] = useState<RecipeWithIngredients>(initialRecipe);
  const [saving, setSaving] = useState(false);
  const [saveStatus, setSaveStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');
  const [errors, setErrors] = useState<Record<string, string>>({});

  // Auto-save debounced function
  const [saveTimeout, setSaveTimeout] = useState<NodeJS.Timeout | null>(null);

  const validateField = (name: string, value: string): string | null => {
    switch (name) {
      case 'title':
        if (!value.trim()) return 'Title is required';
        if (value.length > 200) return 'Title must be less than 200 characters';
        return null;
      case 'servings':
        if (value && value.length > 50) return 'Servings must be less than 50 characters';
        return null;
      case 'instructions':
        if (value && value.length > 10000) return 'Instructions must be less than 10,000 characters';
        return null;
      case 'tips':
        if (value && value.length > 2000) return 'Tips must be less than 2,000 characters';
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
    if (saveTimeout) {
      clearTimeout(saveTimeout);
    }

    // Only auto-save if there are no validation errors
    if (!error) {
      setSaveStatus('saving');
      const newTimeout = setTimeout(async () => {
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

      setSaveTimeout(newTimeout);
    }
  }, [saveTimeout]);

  const saveRecipe = async (updates: Partial<Recipe>) => {
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
      setRecipe(prev => ({ ...prev, ...updatedRecipe }));
    } catch (error) {
      console.error('Failed to save recipe:', error);
      throw error;
    } finally {
      setSaving(false);
    }
  };

  const handlePublish = async () => {
    // Validate all required fields
    const titleError = validateField('title', recipe.title);
    if (titleError) {
      setErrors(prev => ({ ...prev, title: titleError }));
      return;
    }

    try {
      setSaving(true);
      await RecipeAPI.updateRecipeStatus(recipe.id, 'published');
      setRecipe(prev => ({ ...prev, status: 'published' }));
      setSaveStatus('saved');
      setTimeout(() => setSaveStatus('idle'), 2000);
    } catch (error) {
      console.error('Failed to publish recipe:', error);
      setSaveStatus('error');
      setTimeout(() => setSaveStatus('idle'), 3000);
    } finally {
      setSaving(false);
    }
  };

  // Cleanup timeout on unmount
  useEffect(() => {
    return () => {
      if (saveTimeout) {
        clearTimeout(saveTimeout);
      }
    };
  }, [saveTimeout]);

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

        <div className="flex space-x-4">
          {recipe.status === 'review_required' && (
            <button
              type="button"
              onClick={handlePublish}
              disabled={saving || !!errors.title}
              className="inline-flex justify-center py-2 px-4 border border-transparent shadow-sm text-sm font-medium rounded-md text-white bg-green-600 hover:bg-green-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-green-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {saving ? 'Publishing...' : 'Publish Recipe'}
            </button>
          )}
        </div>
      </div>
    </form>
  );
}