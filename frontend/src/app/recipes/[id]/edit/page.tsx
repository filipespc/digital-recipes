'use client';

import { use } from 'react';
import Link from 'next/link';
import { useRecipe } from '@/hooks/useRecipes';
import LoadingSpinner from '@/components/LoadingSpinner';
import RecipeEditForm from '@/components/RecipeEditForm';

interface RecipeEditPageProps {
  params: Promise<{ id: string }>;
}

export default function RecipeEditPage({ params }: RecipeEditPageProps) {
  const resolvedParams = use(params);
  const recipeId = parseInt(resolvedParams.id, 10);
  const { recipe, loading, error } = useRecipe(recipeId);

  if (loading) {
    return (
      <div className="min-h-screen bg-gray-50">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
          <LoadingSpinner size="lg" className="mt-20" />
          <p className="text-center text-gray-600 mt-4">Loading recipe...</p>
        </div>
      </div>
    );
  }

  if (error || !recipe) {
    return (
      <div className="min-h-screen bg-gray-50">
        <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
          <div className="text-center">
            <div className="text-red-600 mb-4">
              <h2 className="text-xl font-semibold">Error loading recipe</h2>
              <p className="mt-2">{error || 'Recipe not found'}</p>
            </div>
            <Link
              href="/recipes"
              className="text-blue-600 hover:text-blue-800 font-medium"
            >
              ← Back to Recipes
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
        <div className="mb-6 flex items-center justify-between">
          <Link
            href={`/recipes/${recipe.id}`}
            className="text-blue-600 hover:text-blue-800 font-medium"
          >
            ← Back to Recipe
          </Link>
          <div className="flex items-center space-x-4">
            <span className="text-sm text-gray-600">
              Status: <span className="font-medium">{recipe.status.replace('_', ' ')}</span>
            </span>
          </div>
        </div>

        <div className="bg-white rounded-lg shadow-sm border border-gray-200">
          <header className="px-6 py-4 border-b border-gray-200">
            <h1 className="text-2xl font-bold text-gray-900">Edit Recipe</h1>
            <p className="mt-1 text-sm text-gray-600">
              Review and edit your recipe details. Changes are saved automatically as you type.
            </p>
          </header>

          <div className="p-6">
            <RecipeEditForm recipe={recipe} />
          </div>
        </div>
      </div>
    </div>
  );
}