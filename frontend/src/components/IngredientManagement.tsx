'use client';

import { useState, useEffect, useCallback } from 'react';
import { IngredientManagement } from '@/types/recipe';
import { RecipeAPI } from '@/services/api';

interface IngredientManagementProps {
  userId?: number; // TODO: Get from authentication context
}

export default function IngredientManagementPage({ userId = 1 }: IngredientManagementProps) {
  const [ingredients, setIngredients] = useState<IngredientManagement[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [editName, setEditName] = useState('');
  const [mergingId, setMergingId] = useState<number | null>(null);
  const [selectedMergeTarget, setSelectedMergeTarget] = useState<number | null>(null);

  const loadIngredients = useCallback(async () => {
    try {
      setLoading(true);
      setError(null);
      const data = await RecipeAPI.getIngredientManagement(userId);
      setIngredients(data);
    } catch (err) {
      console.error('Failed to load ingredients:', err);
      setError('Failed to load ingredients. Please try again.');
    } finally {
      setLoading(false);
    }
  }, [userId]);

  useEffect(() => {
    loadIngredients();
  }, [loadIngredients]);

  const handleRename = async (ingredientId: number, newName: string) => {
    try {
      await RecipeAPI.updatePantryItem(ingredientId, newName.trim(), userId);
      await loadIngredients(); // Refresh the list
      setEditingId(null);
      setEditName('');
    } catch (err) {
      console.error('Failed to rename ingredient:', err);
      setError('Failed to rename ingredient. Please try again.');
    }
  };

  const handleDelete = async (ingredientId: number, ingredientName: string) => {
    if (!confirm(`Are you sure you want to delete "${ingredientName}"? This action cannot be undone.`)) {
      return;
    }

    try {
      await RecipeAPI.deletePantryItem(ingredientId, userId);
      await loadIngredients(); // Refresh the list
    } catch (err) {
      console.error('Failed to delete ingredient:', err);
      setError('Failed to delete ingredient. It may be used in recipes.');
    }
  };

  const handleMerge = async (targetId: number, sourceId: number) => {
    const targetIngredient = ingredients.find(i => i.id === targetId);
    const sourceIngredient = ingredients.find(i => i.id === sourceId);

    if (!targetIngredient || !sourceIngredient) {
      setError('Invalid ingredients selected for merge.');
      return;
    }

    if (!confirm(`Merge "${sourceIngredient.name}" into "${targetIngredient.name}"? All recipes using "${sourceIngredient.name}" will be updated to use "${targetIngredient.name}".`)) {
      return;
    }

    try {
      await RecipeAPI.mergePantryItems(targetId, sourceId, userId);
      await loadIngredients(); // Refresh the list
      setMergingId(null);
      setSelectedMergeTarget(null);
    } catch (err) {
      console.error('Failed to merge ingredients:', err);
      setError('Failed to merge ingredients. Please try again.');
    }
  };

  const handleStartEdit = (ingredient: IngredientManagement) => {
    setEditingId(ingredient.id);
    setEditName(ingredient.name);
  };

  const handleCancelEdit = () => {
    setEditingId(null);
    setEditName('');
  };

  const handleStartMerge = (ingredientId: number) => {
    setMergingId(ingredientId);
    setSelectedMergeTarget(null);
  };

  const handleCancelMerge = () => {
    setMergingId(null);
    setSelectedMergeTarget(null);
  };

  if (loading) {
    return (
      <div className="flex items-center justify-center py-12">
        <div className="animate-spin rounded-full h-8 w-8 border-b-2 border-blue-600"></div>
        <span className="ml-3 text-gray-600">Loading ingredients...</span>
      </div>
    );
  }

  return (
    <div className="max-w-4xl mx-auto py-8 px-4 sm:px-6 lg:px-8">
      <div className="mb-8">
        <h1 className="text-3xl font-bold text-gray-900">Ingredient Management</h1>
        <p className="mt-2 text-gray-600">
          Manage your ingredient collection. Merge duplicates, rename ingredients, and keep your recipes organized.
        </p>
      </div>

      {error && (
        <div className="mb-6 bg-red-50 border border-red-200 rounded-md p-4">
          <div className="flex">
            <svg className="h-5 w-5 text-red-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            </svg>
            <div className="ml-3">
              <p className="text-sm text-red-800">{error}</p>
            </div>
            <button
              onClick={() => setError(null)}
              className="ml-auto pl-3"
            >
              <svg className="h-5 w-5 text-red-400 hover:text-red-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
        </div>
      )}

      {ingredients.length === 0 ? (
        <div className="text-center py-12">
          <svg className="mx-auto h-12 w-12 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
          </svg>
          <h3 className="mt-2 text-sm font-medium text-gray-900">No ingredients yet</h3>
          <p className="mt-1 text-sm text-gray-500">
            Ingredients will appear here when you add and link them in your recipes.
          </p>
        </div>
      ) : (
        <div className="bg-white shadow rounded-lg overflow-hidden">
          <div className="px-4 py-5 sm:p-6">
            <div className="flex items-center justify-between mb-6">
              <h2 className="text-lg font-medium text-gray-900">
                Your Ingredients ({ingredients.length})
              </h2>
              <button
                onClick={loadIngredients}
                className="inline-flex items-center px-3 py-2 border border-gray-300 shadow-sm text-sm leading-4 font-medium rounded-md text-gray-700 bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-blue-500"
              >
                <svg className="h-4 w-4 mr-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15" />
                </svg>
                Refresh
              </button>
            </div>

            <div className="space-y-4">
              {ingredients.map((ingredient) => (
                <IngredientManagementItem
                  key={ingredient.id}
                  ingredient={ingredient}
                  isEditing={editingId === ingredient.id}
                  editName={editName}
                  onEditNameChange={setEditName}
                  onStartEdit={handleStartEdit}
                  onCancelEdit={handleCancelEdit}
                  onSaveEdit={(newName) => handleRename(ingredient.id, newName)}
                  onDelete={() => handleDelete(ingredient.id, ingredient.name)}
                  onStartMerge={() => handleStartMerge(ingredient.id)}
                  isMerging={mergingId === ingredient.id}
                  mergeOptions={ingredients.filter(i => i.id !== ingredient.id)}
                  selectedMergeTarget={selectedMergeTarget}
                  onMergeTargetChange={setSelectedMergeTarget}
                  onCancelMerge={handleCancelMerge}
                  onConfirmMerge={(targetId) => handleMerge(targetId, ingredient.id)}
                />
              ))}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

// Individual ingredient management item component
interface IngredientManagementItemProps {
  ingredient: IngredientManagement;
  isEditing: boolean;
  editName: string;
  onEditNameChange: (name: string) => void;
  onStartEdit: (ingredient: IngredientManagement) => void;
  onCancelEdit: () => void;
  onSaveEdit: (newName: string) => void;
  onDelete: () => void;
  onStartMerge: () => void;
  isMerging: boolean;
  mergeOptions: IngredientManagement[];
  selectedMergeTarget: number | null;
  onMergeTargetChange: (targetId: number | null) => void;
  onCancelMerge: () => void;
  onConfirmMerge: (targetId: number) => void;
}

function IngredientManagementItem({
  ingredient,
  isEditing,
  editName,
  onEditNameChange,
  onStartEdit,
  onCancelEdit,
  onSaveEdit,
  onDelete,
  onStartMerge,
  isMerging,
  mergeOptions,
  selectedMergeTarget,
  onMergeTargetChange,
  onCancelMerge,
  onConfirmMerge,
}: IngredientManagementItemProps) {

  const handleSaveEdit = () => {
    if (editName.trim() && editName.trim() !== ingredient.name) {
      onSaveEdit(editName.trim());
    } else {
      onCancelEdit();
    }
  };

  const handleConfirmMerge = () => {
    if (selectedMergeTarget) {
      onConfirmMerge(selectedMergeTarget);
    }
  };

  return (
    <div className="border border-gray-200 rounded-lg p-4 hover:border-gray-300 transition-colors">
      {isEditing ? (
        <div className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">
              Ingredient Name
            </label>
            <input
              type="text"
              value={editName}
              onChange={(e) => onEditNameChange(e.target.value)}
              className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
              placeholder="Enter ingredient name"
              autoFocus
            />
          </div>
          <div className="flex justify-end space-x-3">
            <button
              onClick={onCancelEdit}
              className="px-3 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              onClick={handleSaveEdit}
              disabled={!editName.trim()}
              className="px-3 py-2 border border-transparent rounded-md text-sm font-medium text-white bg-blue-600 hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Save
            </button>
          </div>
        </div>
      ) : isMerging ? (
        <div className="space-y-3">
          <div>
            <h4 className="font-medium text-gray-900">Merge "{ingredient.name}" into:</h4>
            <p className="text-sm text-gray-600 mb-3">
              Select the ingredient you want to keep. All recipes using "{ingredient.name}" will be updated.
            </p>
            <select
              value={selectedMergeTarget || ''}
              onChange={(e) => onMergeTargetChange(e.target.value ? parseInt(e.target.value) : null)}
              className="block w-full px-3 py-2 border border-gray-300 rounded-md shadow-sm focus:outline-none focus:ring-blue-500 focus:border-blue-500"
            >
              <option value="">Select target ingredient...</option>
              {mergeOptions.map((option) => (
                <option key={option.id} value={option.id}>
                  {option.name} ({option.usage_count} recipes)
                </option>
              ))}
            </select>
          </div>
          <div className="flex justify-end space-x-3">
            <button
              onClick={onCancelMerge}
              className="px-3 py-2 border border-gray-300 rounded-md text-sm font-medium text-gray-700 bg-white hover:bg-gray-50"
            >
              Cancel
            </button>
            <button
              onClick={handleConfirmMerge}
              disabled={!selectedMergeTarget}
              className="px-3 py-2 border border-transparent rounded-md text-sm font-medium text-white bg-red-600 hover:bg-red-700 disabled:opacity-50 disabled:cursor-not-allowed"
            >
              Merge Ingredients
            </button>
          </div>
        </div>
      ) : (
        <div className="flex items-center justify-between">
          <div className="flex-1">
            <div className="flex items-center space-x-3">
              <h3 className="text-lg font-medium text-gray-900">{ingredient.name}</h3>
              {ingredient.is_approved && (
                <span className="inline-flex items-center px-2 py-1 rounded-full text-xs font-medium bg-green-100 text-green-800">
                  Approved
                </span>
              )}
            </div>
            <div className="mt-1 flex items-center space-x-4 text-sm text-gray-600">
              <span>Used in {ingredient.usage_count} recipe{ingredient.usage_count !== 1 ? 's' : ''}</span>
              <span>•</span>
              <span>Created {new Date(ingredient.created_at).toLocaleDateString()}</span>
            </div>
          </div>

          <div className="flex items-center space-x-2">
            <button
              onClick={() => onStartEdit(ingredient)}
              className="text-blue-600 hover:text-blue-800 text-sm font-medium"
            >
              Rename
            </button>
            {mergeOptions.length > 0 && (
              <button
                onClick={onStartMerge}
                className="text-yellow-600 hover:text-yellow-800 text-sm font-medium"
              >
                Merge
              </button>
            )}
            <button
              onClick={onDelete}
              disabled={ingredient.usage_count > 0}
              className="text-red-600 hover:text-red-800 text-sm font-medium disabled:opacity-50 disabled:cursor-not-allowed"
              title={ingredient.usage_count > 0 ? 'Cannot delete ingredients used in recipes' : 'Delete ingredient'}
            >
              Delete
            </button>
          </div>
        </div>
      )}
    </div>
  );
}