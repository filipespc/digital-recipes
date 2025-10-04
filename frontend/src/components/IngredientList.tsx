'use client';

import { useState, useCallback } from 'react';
import { RecipeIngredient, PantryItem } from '@/types/recipe';
import { RecipeAPI } from '@/services/api';
import IngredientStateToggle from './IngredientStateToggle';

interface IngredientListProps {
  recipeId: number;
  ingredients: RecipeIngredient[];
  onIngredientsChange: (ingredients: RecipeIngredient[]) => void;
}

export default function IngredientList({
  recipeId,
  ingredients,
  onIngredientsChange
}: IngredientListProps) {
  const [editingId, setEditingId] = useState<number | null>(null);
  const [showAddForm, setShowAddForm] = useState(false);
  const [loading, setLoading] = useState<Record<number | string, boolean>>({});
  const [errors, setErrors] = useState<Record<number | string, string>>({});

  // New ingredient form state
  const [newIngredient, setNewIngredient] = useState({
    original_text: '',
    quantity: '',
    unit: ''
  });

  const validateIngredient = (ingredient: { original_text: string; quantity: string; unit: string }): string | null => {
    const trimmed = ingredient.original_text.trim();

    if (!trimmed) return 'Ingredient text is required';
    if (trimmed.length < 2) return 'Ingredient text must be at least 2 characters';
    if (trimmed.length > 200) return 'Ingredient text must be less than 200 characters';

    // Security checks - detect potential script injection attempts
    const dangerousPatterns = [
      /<script[^>]*>/i,
      /javascript:/i,
      /data:text\/html/i,
      /on\w+\s*=/i,
      /<iframe[^>]*>/i,
      /<object[^>]*>/i,
      /<embed[^>]*>/i
    ];

    for (const pattern of dangerousPatterns) {
      if (pattern.test(ingredient.original_text)) {
        return 'Invalid content detected. HTML/JavaScript is not allowed.';
      }
    }

    // Validate quantity if provided
    if (ingredient.quantity && ingredient.quantity.trim()) {
      const qty = parseFloat(ingredient.quantity);
      if (isNaN(qty) || qty <= 0) {
        return 'Quantity must be a positive number';
      }
    }

    // Validate unit if provided
    if (ingredient.unit && ingredient.unit.trim().length > 50) {
      return 'Unit must be less than 50 characters';
    }

    return null;
  };

  const handleAddIngredient = useCallback(async () => {
    const error = validateIngredient(newIngredient);
    if (error) {
      setErrors(prev => ({ ...prev, new: error }));
      return;
    }

    setLoading(prev => ({ ...prev, new: true }));
    setErrors(prev => ({ ...prev, new: '' }));

    try {
      const ingredientData = {
        original_text: newIngredient.original_text.trim(),
        quantity: newIngredient.quantity.trim() ? parseFloat(newIngredient.quantity) : undefined,
        unit: newIngredient.unit.trim() || undefined
      };

      const createdIngredient = await RecipeAPI.addIngredient(recipeId, ingredientData);

      onIngredientsChange([...ingredients, createdIngredient]);
      setNewIngredient({ original_text: '', quantity: '', unit: '' });
      setShowAddForm(false);
    } catch (error) {
      console.error('Failed to add ingredient:', error);
      setErrors(prev => ({ ...prev, new: 'Failed to add ingredient. Please try again.' }));
    } finally {
      setLoading(prev => ({ ...prev, new: false }));
    }
  }, [recipeId, ingredients, newIngredient, onIngredientsChange]);

  const handleUpdateIngredient = useCallback(async (ingredientId: number, updates: Partial<RecipeIngredient>) => {
    setLoading(prev => ({ ...prev, [ingredientId]: true }));
    setErrors(prev => ({ ...prev, [ingredientId]: '' }));

    try {
      const updateData: any = {};
      if (updates.original_text !== undefined) updateData.original_text = updates.original_text;
      if (updates.quantity !== undefined) updateData.quantity = updates.quantity;
      if (updates.unit !== undefined) updateData.unit = updates.unit;

      const updatedIngredient = await RecipeAPI.updateIngredient(recipeId, ingredientId, updateData);

      const updatedIngredients = ingredients.map(ing =>
        ing.id === ingredientId ? updatedIngredient : ing
      );
      onIngredientsChange(updatedIngredients);
      setEditingId(null);
    } catch (error) {
      console.error('Failed to update ingredient:', error);
      setErrors(prev => ({ ...prev, [ingredientId]: 'Failed to update ingredient. Please try again.' }));
    } finally {
      setLoading(prev => ({ ...prev, [ingredientId]: false }));
    }
  }, [recipeId, ingredients, onIngredientsChange]);

  const handleDeleteIngredient = useCallback(async (ingredientId: number) => {
    if (!confirm('Are you sure you want to delete this ingredient?')) return;

    setLoading(prev => ({ ...prev, [ingredientId]: true }));

    try {
      await RecipeAPI.deleteIngredient(recipeId, ingredientId);

      const updatedIngredients = ingredients.filter(ing => ing.id !== ingredientId);
      onIngredientsChange(updatedIngredients);
    } catch (error) {
      console.error('Failed to delete ingredient:', error);
      setErrors(prev => ({ ...prev, [ingredientId]: 'Failed to delete ingredient. Please try again.' }));
    } finally {
      setLoading(prev => ({ ...prev, [ingredientId]: false }));
    }
  }, [recipeId, ingredients, onIngredientsChange]);

  const handleLinkToPantryItem = useCallback(async (ingredientId: number, pantryItem: PantryItem) => {
    setLoading(prev => ({ ...prev, [ingredientId]: true }));

    try {
      const updatedIngredient = await RecipeAPI.linkIngredientToPantryItem(
        recipeId,
        ingredientId,
        pantryItem.id
      );

      // Find the original ingredient to preserve its data
      const originalIngredient = ingredients.find(ing => ing.id === ingredientId);

      // Merge the updated data with the original ingredient data to preserve all fields
      const completeUpdatedIngredient = {
        ...originalIngredient, // Keep all original fields
        ...updatedIngredient,   // Apply updates from API
        pantry_item_name: pantryItem.name,
        pantry_item_id: pantryItem.id
      };

      const updatedIngredients = ingredients.map(ing =>
        ing.id === ingredientId ? completeUpdatedIngredient : ing
      );
      onIngredientsChange(updatedIngredients);
    } catch (error) {
      console.error('Failed to link ingredient to pantry item:', error);
      setErrors(prev => ({ ...prev, [ingredientId]: 'Failed to link ingredient to pantry item. Please try again.' }));
    } finally {
      setLoading(prev => ({ ...prev, [ingredientId]: false }));
    }
  }, [recipeId, ingredients, onIngredientsChange]);

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h3 className="text-lg font-medium text-gray-900">Ingredients</h3>
        <button
          onClick={() => setShowAddForm(true)}
          className="inline-flex items-center px-3 py-2 border border-transparent text-sm leading-4 font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
        >
          <svg className="h-4 w-4 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
          </svg>
          Add Ingredient
        </button>
      </div>

      {/* Existing Ingredients List */}
      <div className="space-y-3">
        {ingredients.map((ingredient) => {
          return (
            <IngredientItem
              key={ingredient.id}
              ingredient={ingredient}
              recipeId={recipeId}
              isEditing={editingId === ingredient.id}
              loading={loading[ingredient.id] || false}
              error={errors[ingredient.id]}
              onEdit={() => setEditingId(ingredient.id)}
              onCancelEdit={() => setEditingId(null)}
              onUpdate={handleUpdateIngredient}
              onDelete={handleDeleteIngredient}
              onLinkToPantryItem={handleLinkToPantryItem}
            />
          );
        })}
      </div>

      {/* Add New Ingredient Form */}
      {showAddForm && (
        <div className="border border-gray-200 rounded-lg p-4 bg-gray-50">
          <h4 className="font-medium text-gray-900 mb-3">Add New Ingredient</h4>

          <div className="space-y-3">
            <div>
              <label className="block text-sm font-medium text-gray-700">
                Ingredient Text *
              </label>
              <input
                type="text"
                value={newIngredient.original_text}
                onChange={(e) => setNewIngredient(prev => ({ ...prev, original_text: e.target.value }))}
                className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                placeholder="e.g., 2 cups all-purpose flour"
              />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Quantity
                </label>
                <input
                  type="number"
                  step="any"
                  value={newIngredient.quantity}
                  onChange={(e) => setNewIngredient(prev => ({ ...prev, quantity: e.target.value }))}
                  className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                  placeholder="2"
                />
              </div>

              <div>
                <label className="block text-sm font-medium text-gray-700">
                  Unit
                </label>
                <input
                  type="text"
                  value={newIngredient.unit}
                  onChange={(e) => setNewIngredient(prev => ({ ...prev, unit: e.target.value }))}
                  className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
                  placeholder="cups"
                />
              </div>
            </div>

            {errors.new && (
              <p className="text-sm text-red-600">{errors.new}</p>
            )}

            <div className="flex justify-end space-x-3">
              <button
                onClick={() => {
                  setShowAddForm(false);
                  setNewIngredient({ original_text: '', quantity: '', unit: '' });
                  setErrors(prev => ({ ...prev, new: '' }));
                }}
                className="px-3 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                Cancel
              </button>
              <button
                onClick={handleAddIngredient}
                disabled={loading.new}
                className="inline-flex items-center px-3 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                {loading.new ? (
                  <>
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                    Adding...
                  </>
                ) : (
                  'Add Ingredient'
                )}
              </button>
            </div>
          </div>
        </div>
      )}

      {ingredients.length === 0 && !showAddForm && (
        <div className="text-center py-8 text-gray-500">
          <p>No ingredients added yet.</p>
          <p className="text-sm mt-1">Click "Add Ingredient" to get started.</p>
        </div>
      )}
    </div>
  );
}

// Individual Ingredient Item Component
interface IngredientItemProps {
  ingredient: RecipeIngredient;
  recipeId: number;
  isEditing: boolean;
  loading: boolean;
  error?: string;
  onEdit: () => void;
  onCancelEdit: () => void;
  onUpdate: (id: number, updates: Partial<RecipeIngredient>) => void;
  onDelete: (id: number) => void;
  onLinkToPantryItem: (id: number, pantryItem: PantryItem) => void;
}

function IngredientItem({
  ingredient,
  recipeId,
  isEditing,
  loading,
  error,
  onEdit,
  onCancelEdit,
  onUpdate,
  onDelete,
  onLinkToPantryItem
}: IngredientItemProps) {
  const [editForm, setEditForm] = useState({
    original_text: ingredient.original_text,
    quantity: ingredient.quantity?.toString() || '',
    unit: ingredient.unit || ''
  });

  const handleSave = () => {
    const updates: Partial<RecipeIngredient> = {
      original_text: editForm.original_text.trim(),
      quantity: editForm.quantity.trim() ? parseFloat(editForm.quantity) : undefined,
      unit: editForm.unit.trim() || undefined
    };
    onUpdate(ingredient.id, updates);
  };

  if (isEditing) {
    return (
      <div className="border border-blue-200 rounded-lg p-4 bg-blue-50">
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700">
              Ingredient Text *
            </label>
            <input
              type="text"
              value={editForm.original_text}
              onChange={(e) => setEditForm(prev => ({ ...prev, original_text: e.target.value }))}
              className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
            />
          </div>

          <div className="grid grid-cols-2 gap-3">
            <div>
              <label className="block text-sm font-medium text-gray-700">
                Quantity
              </label>
              <input
                type="number"
                step="any"
                value={editForm.quantity}
                onChange={(e) => setEditForm(prev => ({ ...prev, quantity: e.target.value }))}
                className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
              />
            </div>

            <div>
              <label className="block text-sm font-medium text-gray-700">
                Unit
              </label>
              <input
                type="text"
                value={editForm.unit}
                onChange={(e) => setEditForm(prev => ({ ...prev, unit: e.target.value }))}
                className="mt-1 block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500 sm:text-sm"
              />
            </div>
          </div>

          {error && (
            <p className="text-sm text-red-600">{error}</p>
          )}

          <div className="flex justify-end space-x-3">
            <button
              onClick={onCancelEdit}
              className="px-3 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
            >
              Cancel
            </button>
            <button
              onClick={handleSave}
              disabled={loading}
              className="inline-flex items-center px-3 py-2 border border-transparent text-sm font-medium rounded-md text-white bg-blue-600 hover:bg-blue-700 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              {loading ? (
                <>
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-white mr-2"></div>
                  Saving...
                </>
              ) : (
                'Save'
              )}
            </button>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center space-x-2">
            <span className="text-sm font-medium text-gray-900">
              {ingredient.original_text}
            </span>
            {ingredient.pantry_item_id && (
              <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                <svg className="h-3 w-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                </svg>
                Linked to Pantry
              </span>
            )}
          </div>

          {(ingredient.quantity || ingredient.unit) && (
            <div className="mt-1 text-sm text-gray-600">
              {ingredient.quantity && <span>{ingredient.quantity}</span>}
              {ingredient.unit && <span className="ml-1">{ingredient.unit}</span>}
            </div>
          )}

          {/* Pantry Item State Toggle */}
          <div className="mt-2">
            <IngredientStateToggle
              recipeId={recipeId}
              ingredient={ingredient}
              onLinkToPantryItem={onLinkToPantryItem}
            />
          </div>
        </div>

        <div className="flex items-center space-x-2 ml-4">
          <button
            onClick={onEdit}
            disabled={loading}
            className="text-blue-600 hover:text-blue-800 text-sm font-medium disabled:opacity-50"
          >
            Edit
          </button>
          <button
            onClick={() => onDelete(ingredient.id)}
            disabled={loading}
            className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50"
          >
            {loading ? 'Deleting...' : 'Delete'}
          </button>
        </div>
      </div>

      {error && (
        <p className="mt-2 text-sm text-red-600">{error}</p>
      )}
    </div>
  );
}