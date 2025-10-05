'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { RecipeIngredient, PantryItem } from '@/types/recipe';
import { RecipeAPI } from '@/services/api';

interface IngredientSearchAutocompleteProps {
  recipeId: number;
  ingredient: RecipeIngredient;
  onLinkToPantryItem: (ingredientId: number, pantryItem: PantryItem) => void;
}

export default function IngredientSearchAutocomplete({
  recipeId,
  ingredient,
  onLinkToPantryItem
}: IngredientSearchAutocompleteProps) {
  const [isOpen, setIsOpen] = useState(false);
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<PantryItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [showCreateForm, setShowCreateForm] = useState(false);
  const [newIngredientName, setNewIngredientName] = useState('');
  const [createLoading, setCreateLoading] = useState(false);

  const searchTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsOpen(false);
        setShowCreateForm(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      if (searchTimeoutRef.current) {
        clearTimeout(searchTimeoutRef.current);
      }
    };
  }, []);

  const searchIngredients = useCallback(async (query: string) => {
    if (!query.trim()) {
      setSearchResults([]);
      return;
    }

    setLoading(true);
    try {
      // TODO: Replace hardcoded user_id with actual user authentication
      const results = await RecipeAPI.searchPantryItems(query, 1);
      setSearchResults(results);
    } catch (error) {
      console.error('Failed to search ingredients:', error);
      setSearchResults([]);
    } finally {
      setLoading(false);
    }
  }, []);

  const handleSearchChange = (value: string) => {
    setSearchQuery(value);
    setShowCreateForm(false);

    // Clear existing timeout
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }

    // Debounce search
    searchTimeoutRef.current = setTimeout(() => {
      searchIngredients(value);
    }, 300);
  };

  const handleSelectPantryItem = (pantryItem: PantryItem) => {
    onLinkToPantryItem(ingredient.id, pantryItem);
    setIsOpen(false);
    setSearchQuery('');
    setSearchResults([]);
  };

  const handleCreateCanonical = async () => {
    if (!newIngredientName.trim()) return;

    setCreateLoading(true);
    try {
      // TODO: Replace hardcoded user_id with actual user authentication
      const created = await RecipeAPI.createPantryItem(newIngredientName.trim(), 1);
      onLinkToPantryItem(ingredient.id, created);
      setIsOpen(false);
      setShowCreateForm(false);
      setNewIngredientName('');
      setSearchQuery('');
    } catch (error) {
      console.error('Failed to create pantry item:', error);
    } finally {
      setCreateLoading(false);
    }
  };

  const showCreateOption = searchQuery.trim() &&
    searchResults.length === 0 &&
    !loading &&
    !ingredient.pantry_item_id;

  // Show different UI based on link status
  if (ingredient.pantry_item_id) {
    return (
      <div className="flex items-center justify-between mt-2">
        <div className="flex items-center space-x-2 text-green-700 text-sm">
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
          <span>Linked to pantry item</span>
        </div>
        <button
          onClick={() => {
            // TODO: Implement unlink functionality
            console.log('Unlink ingredient:', ingredient.id);
          }}
          className="text-blue-600 hover:text-blue-800 text-xs font-medium"
        >
          Change
        </button>
      </div>
    );
  }

  return (
    <div className="relative" ref={dropdownRef}>
      <div className="flex items-center justify-between">
        <div className="flex items-center space-x-2 text-yellow-700 text-sm">
          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-2.5L13.732 4c-.77-.833-1.732-.833-2.5 0L4.268 16.5c-.77.833.192 2.5 1.732 2.5z" />
          </svg>
          <span>Needs Resolution</span>
        </div>
        <button
          onClick={() => setIsOpen(!isOpen)}
          className="inline-flex items-center px-2 py-1 border border-orange-300 rounded text-xs font-medium text-orange-700 bg-orange-50 hover:bg-orange-100 focus:outline-none focus:ring-2 focus:ring-offset-2 focus:ring-orange-500"
        >
          <svg className="h-3 w-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.102m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
          </svg>
          Resolve
        </button>
      </div>

      {isOpen && (
        <div className="absolute z-10 mt-1 w-80 bg-white shadow-lg border border-gray-200 rounded-md py-1 max-h-64 overflow-auto">
          {/* Search Input */}
          <div className="px-3 py-2 border-b border-gray-200">
            <input
              type="text"
              value={searchQuery}
              onChange={(e) => handleSearchChange(e.target.value)}
              placeholder="Search for pantry items..."
              className="w-full px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500"
              autoFocus
            />
          </div>

          {/* Loading State */}
          {loading && (
            <div className="px-3 py-2 text-sm text-gray-500 flex items-center">
              <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600 mr-2"></div>
              Searching...
            </div>
          )}

          {/* Search Results */}
          {!loading && searchResults.length > 0 && (
            <div>
              <div className="px-3 py-1 text-xs font-medium text-gray-500 border-b border-gray-100">
                Found {searchResults.length} pantry item{searchResults.length !== 1 ? 's' : ''}
              </div>
              {searchResults.map((pantryItem) => (
                <button
                  key={pantryItem.id}
                  onClick={() => handleSelectPantryItem(pantryItem)}
                  className="w-full px-3 py-2 text-left text-sm hover:bg-gray-50 flex items-center justify-between"
                >
                  <div>
                    <div className="font-medium text-gray-900">{pantryItem.name}</div>
                    {pantryItem.is_approved && (
                      <div className="text-xs text-green-600 flex items-center mt-1">
                        <svg className="h-3 w-3 mr-1" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
                        </svg>
                        Approved
                      </div>
                    )}
                  </div>
                  <svg className="h-4 w-4 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                    <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.102m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                  </svg>
                </button>
              ))}
            </div>
          )}

          {/* No Results - Show Create Option */}
          {showCreateOption && (
            <div>
              <div className="px-3 py-1 text-xs font-medium text-gray-500 border-b border-gray-100">
                No matching pantry items found
              </div>
              <button
                onClick={() => {
                  setShowCreateForm(true);
                  setNewIngredientName(searchQuery);
                }}
                className="w-full px-3 py-2 text-left text-sm hover:bg-gray-50 flex items-center"
              >
                <svg className="h-4 w-4 mr-2 text-blue-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M12 6v6m0 0v6m0-6h6m-6 0H6" />
                </svg>
                <span className="text-blue-600">Create new pantry item</span>
              </button>
            </div>
          )}

          {/* Create New Canonical Ingredient Form */}
          {showCreateForm && (
            <div className="border-t border-gray-200 px-3 py-3">
              <div className="text-xs font-medium text-gray-700 mb-2">
                Create New Pantry Item
              </div>
              <input
                type="text"
                value={newIngredientName}
                onChange={(e) => setNewIngredientName(e.target.value)}
                placeholder="Enter ingredient name"
                className="w-full px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-blue-500 focus:border-blue-500 mb-2"
              />
              <div className="flex justify-end space-x-2">
                <button
                  onClick={() => {
                    setShowCreateForm(false);
                    setNewIngredientName('');
                  }}
                  className="px-2 py-1 text-xs text-gray-600 hover:text-gray-800"
                >
                  Cancel
                </button>
                <button
                  onClick={handleCreateCanonical}
                  disabled={createLoading || !newIngredientName.trim()}
                  className="inline-flex items-center px-2 py-1 bg-blue-600 text-white text-xs rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  {createLoading ? (
                    <>
                      <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                      Creating...
                    </>
                  ) : (
                    'Create & Link'
                  )}
                </button>
              </div>
            </div>
          )}

          {/* Empty State */}
          {!loading && !searchQuery.trim() && (
            <div className="px-3 py-4 text-sm text-gray-500 text-center">
              <svg className="mx-auto h-8 w-8 text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z" />
              </svg>
              Start typing to search for pantry items
            </div>
          )}
        </div>
      )}
    </div>
  );
}