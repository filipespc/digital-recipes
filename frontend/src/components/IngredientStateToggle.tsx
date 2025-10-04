'use client';

import { useState, useEffect, useRef, useCallback } from 'react';
import { RecipeIngredient, PantryItem } from '@/types/recipe';
import { RecipeAPI } from '@/services/api';

interface IngredientStateToggleProps {
  recipeId: number;
  ingredient: RecipeIngredient;
  onLinkToPantryItem: (ingredientId: number, pantryItem: PantryItem) => void;
}

type IngredientState = 'new' | 'linked';

export default function IngredientStateToggle({
  recipeId,
  ingredient,
  onLinkToPantryItem
}: IngredientStateToggleProps) {
  // Determine initial state based on whether ingredient is linked
  const [state, setState] = useState<IngredientState>(
    ingredient.pantry_item_id ? 'linked' : 'new'
  );

  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState<PantryItem[]>([]);
  const [loading, setLoading] = useState(false);
  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const [newPantryItemName, setNewPantryItemName] = useState('');
  const [loadingSuggestion, setLoadingSuggestion] = useState(false);
  const [selectedPantryItem, setSelectedPantryItem] = useState<PantryItem | null>(null);

  const searchTimeoutRef = useRef<NodeJS.Timeout | null>(null);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const searchPantryItems = useCallback(async (query: string) => {
    setLoading(true);
    try {
      // If query is empty, load all pantry items; otherwise search
      const results = await RecipeAPI.searchPantryItems(query || '', 1);
      setSearchResults(results);
    } catch (error) {
      console.error('Failed to search pantry items:', error);
      setSearchResults([]);
    } finally {
      setLoading(false);
    }
  }, []);

  // Load AI-powered pantry name suggestion when in 'new' state
  useEffect(() => {
    if (state === 'new' && !newPantryItemName && !loadingSuggestion) {
      setLoadingSuggestion(true);

      RecipeAPI.suggestPantryItemName(ingredient.original_text)
        .then((suggestion) => {
          setNewPantryItemName(suggestion.suggested_name);
        })
        .catch((error) => {
          console.error('Failed to get AI pantry suggestion:', error);
          // Fallback to regex extraction
          setNewPantryItemName(extractPantryItemName(ingredient.original_text));
        })
        .finally(() => {
          setLoadingSuggestion(false);
        });
    }
  }, [state, ingredient.original_text, newPantryItemName, loadingSuggestion]);

  // Load all pantry items when dropdown opens
  useEffect(() => {
    if (isDropdownOpen && searchResults.length === 0 && !searchQuery) {
      // Load all pantry items initially
      searchPantryItems('');
    }
  }, [isDropdownOpen, searchPantryItems, searchResults.length, searchQuery]);

  // Close dropdown when clicking outside
  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false);
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

  const handleSearchChange = (value: string) => {
    setSearchQuery(value);

    // Clear existing timeout
    if (searchTimeoutRef.current) {
      clearTimeout(searchTimeoutRef.current);
    }

    // Debounce search
    searchTimeoutRef.current = setTimeout(() => {
      searchPantryItems(value);
    }, 300);
  };

  const handleSelectPantryItem = (pantryItem: PantryItem) => {
    setSelectedPantryItem(pantryItem);
    onLinkToPantryItem(ingredient.id, pantryItem);
    setIsDropdownOpen(false);
    setSearchQuery('');
    setSearchResults([]);
  };

  const handleCreateNewPantryItem = async () => {
    if (!newPantryItemName.trim()) return;

    setLoading(true);
    try {
      const created = await RecipeAPI.createPantryItem(newPantryItemName.trim(), 1);
      onLinkToPantryItem(ingredient.id, created);
    } catch (error) {
      console.error('Failed to create pantry item:', error);
    } finally {
      setLoading(false);
    }
  };

  const toggleState = (e: React.MouseEvent) => {
    e.preventDefault();
    e.stopPropagation();

    setState(state === 'new' ? 'linked' : 'new');
    if (state === 'linked') {
      // When switching from linked to new, reset name and let useEffect handle AI suggestion
      setNewPantryItemName('');
    } else {
      // When switching from new to linked, open the search dropdown
      setIsDropdownOpen(true);
      setSearchQuery('');
    }
  };

  if (state === 'new') {
    return (
      <div className="mt-2 p-3 bg-blue-50 border border-blue-200 rounded-md">
        <div className="flex items-start justify-between">
          <div className="flex-1">
            <div className="flex items-center space-x-2 text-blue-700 text-sm mb-2">
              <span className="text-lg">🆕</span>
              <span className="font-medium">Will auto-create when publishing:</span>
            </div>
            <div className="relative">
              <input
                type="text"
                value={newPantryItemName}
                onChange={(e) => setNewPantryItemName(e.target.value)}
                disabled={loadingSuggestion}
                className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm focus:outline-none focus:ring-2 focus:ring-blue-500 focus:border-blue-500 disabled:bg-gray-100 disabled:cursor-not-allowed"
                placeholder={loadingSuggestion ? "Generating AI suggestion..." : "Enter pantry item name"}
              />
              {loadingSuggestion && (
                <div className="absolute right-3 top-1/2 transform -translate-y-1/2">
                  <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-blue-600"></div>
                </div>
              )}
            </div>
          </div>
          <button
            type="button"
            onClick={toggleState}
            className="ml-3 px-3 py-1 text-xs font-medium text-blue-600 bg-white border border-blue-300 rounded hover:bg-blue-50 focus:outline-none focus:ring-2 focus:ring-blue-500"
          >
            Link to existing instead
          </button>
        </div>
        {newPantryItemName.trim() && (
          <div className="mt-2 flex justify-end">
            <button
              type="button"
              onClick={handleCreateNewPantryItem}
              disabled={loading}
              className="inline-flex items-center px-3 py-1 text-xs font-medium text-white bg-blue-600 rounded hover:bg-blue-700 disabled:opacity-50 disabled:cursor-not-allowed focus:outline-none focus:ring-2 focus:ring-blue-500"
            >
              {loading ? (
                <>
                  <div className="animate-spin rounded-full h-3 w-3 border-b-2 border-white mr-1"></div>
                  Creating...
                </>
              ) : (
                'Create Pantry Item'
              )}
            </button>
          </div>
        )}
      </div>
    );
  }

  // Linked state
  const currentPantryItem = selectedPantryItem || (ingredient.pantry_item_id && ingredient.pantry_item_name ? {
    id: ingredient.pantry_item_id,
    name: ingredient.pantry_item_name,
    user_id: 1,
    created_at: '',
    updated_at: ''
  } : null);

  return (
    <div className="mt-2 p-3 bg-green-50 border border-green-200 rounded-md">
      <div className="flex items-start justify-between">
        <div className="flex-1">
          <div className="flex items-center space-x-2 text-green-700 text-sm mb-2">
            <span className="text-lg">🔗</span>
            <span className="font-medium">
              "{ingredient.original_text}" → linked to pantry item
              {currentPantryItem && <span className="font-semibold"> "{currentPantryItem.name}"</span>}
            </span>
          </div>

          <div className="relative" ref={dropdownRef}>
            <button
              type="button"
              onClick={() => setIsDropdownOpen(!isDropdownOpen)}
              className="w-full px-3 py-2 border border-gray-300 rounded-md text-sm text-left bg-white hover:bg-gray-50 focus:outline-none focus:ring-2 focus:ring-green-500 focus:border-green-500 flex items-center justify-between"
            >
              <span className="text-gray-700">
                {currentPantryItem ? currentPantryItem.name : 'Select pantry item...'}
              </span>
              <svg className="h-4 w-4 text-gray-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
              </svg>
            </button>

            {isDropdownOpen && (
              <div className="absolute z-20 mt-1 w-full bg-white shadow-lg border border-gray-200 rounded-md py-1 max-h-64 overflow-auto">
                {/* Search Input */}
                <div className="px-3 py-2 border-b border-gray-200">
                  <input
                    type="text"
                    value={searchQuery}
                    onChange={(e) => handleSearchChange(e.target.value)}
                    placeholder="Type to filter pantry items..."
                    className="w-full px-2 py-1 border border-gray-300 rounded text-sm focus:outline-none focus:ring-1 focus:ring-green-500 focus:border-green-500"
                    autoFocus
                  />
                </div>

                {/* Loading State */}
                {loading && (
                  <div className="px-3 py-2 text-sm text-gray-500 flex items-center">
                    <div className="animate-spin rounded-full h-4 w-4 border-b-2 border-green-600 mr-2"></div>
                    Loading pantry items...
                  </div>
                )}

                {/* Search Results */}
                {!loading && searchResults.length > 0 && (
                  <div>
                    <div className="px-3 py-1 text-xs font-medium text-gray-500 border-b border-gray-100">
                      {searchQuery ? `Filtered: ${searchResults.length} item${searchResults.length !== 1 ? 's' : ''}` : `All pantry items (${searchResults.length})`}
                    </div>
                    {searchResults.map((pantryItem) => (
                      <button
                        key={pantryItem.id}
                        type="button"
                        onClick={() => handleSelectPantryItem(pantryItem)}
                        className="w-full px-3 py-2 text-left text-sm hover:bg-gray-50 flex items-center justify-between"
                      >
                        <div>
                          <div className="font-medium text-gray-900">{pantryItem.name}</div>
                        </div>
                        <svg className="h-4 w-4 text-green-600" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                          <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.102m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" />
                        </svg>
                      </button>
                    ))}
                  </div>
                )}

                {/* No Results */}
                {!loading && searchQuery.trim() && searchResults.length === 0 && (
                  <div className="px-3 py-4 text-sm text-gray-500 text-center">
                    No pantry items found for "{searchQuery}"
                  </div>
                )}

                {/* Empty State - only show if really no items */}
                {!loading && !searchQuery && searchResults.length === 0 && (
                  <div className="px-3 py-4 text-sm text-gray-500 text-center">
                    <svg className="mx-auto h-6 w-6 text-gray-400 mb-2" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10" />
                    </svg>
                    No pantry items yet. Start by creating one!
                  </div>
                )}
              </div>
            )}
          </div>
        </div>

        <button
          type="button"
          onClick={toggleState}
          className="ml-3 px-3 py-1 text-xs font-medium text-green-600 bg-white border border-green-300 rounded hover:bg-green-50 focus:outline-none focus:ring-2 focus:ring-green-500"
        >
          Create new instead
        </button>
      </div>
    </div>
  );
}

// Helper function to extract a clean pantry item name from ingredient text
// This will be enhanced when we add AI suggestions
function extractPantryItemName(originalText: string): string {
  // Simple extraction logic - remove quantities and common descriptors
  let name = originalText
    .replace(/^\d+(\.\d+)?\s*/, '') // Remove leading numbers like "1 ", "2.5 "
    .replace(/^\d+\/\d+\s*/, '') // Remove fractions like "1/2 "
    .replace(/\b(cups?|tbsp|tsp|oz|lbs?|grams?|kg|ml|liters?|g|l)\b/gi, ' ') // Remove English units
    .replace(/\b(colheres?|colher|chá|sopa|xícaras?|xícara|gramas?|quilos?|litros?)\b/gi, ' ') // Remove Portuguese units
    .replace(/\([^)]*\)/g, ' ') // Remove anything in parentheses
    .replace(/\b(large|small|medium|fresh|dried|chopped|diced|sliced|grande|pequeno|pequena|médio|média|fresco|fresca|seco|seca|picado|picada|cortado|cortada|dente|dentes|moída|moído|na hora)\b/gi, ' ') // Remove descriptors (English + Portuguese)
    .replace(/[,\.]\s*$/, '') // Remove trailing punctuation
    .replace(/\s+/g, ' ') // Normalize multiple spaces to single space
    .trim();

  // If result is empty, return the original text
  if (!name) {
    name = originalText;
  }

  // Capitalize first letter, keep the rest as-is to preserve proper nouns
  return name.charAt(0).toUpperCase() + name.slice(1);
}