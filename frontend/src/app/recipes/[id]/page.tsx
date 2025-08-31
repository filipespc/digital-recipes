'use client';

import { use } from 'react';
import Link from 'next/link';
import { useRecipe } from '@/hooks/useRecipes';
import LoadingSpinner from '@/components/LoadingSpinner';

interface RecipeDetailPageProps {
  params: Promise<{ id: string }>;
}

export default function RecipeDetailPage({ params }: RecipeDetailPageProps) {
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

  const getStatusColor = (status: string) => {
    switch (status) {
      case 'published':
        return 'bg-green-100 text-green-800';
      case 'review_required':
        return 'bg-yellow-100 text-yellow-800';
      case 'processing':
        return 'bg-blue-100 text-blue-800';
      case 'failed':
        return 'bg-red-100 text-red-800';
      default:
        return 'bg-gray-100 text-gray-800';
    }
  };

  return (
    <div className="min-h-screen bg-gray-50">
      <div className="max-w-4xl mx-auto px-4 sm:px-6 lg:px-8 py-10">
        <div className="mb-6">
          <Link
            href="/recipes"
            className="text-blue-600 hover:text-blue-800 font-medium"
          >
            ← Back to Recipes
          </Link>
        </div>

        <article className="bg-white rounded-lg shadow-sm border border-gray-200">
          <header className="px-6 py-6 border-b border-gray-200">
            <div className="flex items-start justify-between">
              <div>
                <h1 className="text-3xl font-bold text-gray-900">{recipe.title}</h1>
                {recipe.servings && (
                  <p className="mt-2 text-gray-600">
                    <span className="font-medium">Servings:</span> {recipe.servings}
                  </p>
                )}
              </div>
              <span
                className={`inline-flex items-center px-3 py-1 rounded-full text-sm font-medium ${getStatusColor(
                  recipe.status
                )}`}
              >
                {recipe.status.replace('_', ' ')}
              </span>
            </div>
          </header>

          <div className="px-6 py-6">
            {recipe.ingredients && recipe.ingredients.length > 0 && (
              <section className="mb-8">
                <h2 className="text-xl font-semibold text-gray-900 mb-4">Ingredients</h2>
                <ul className="space-y-2">
                  {recipe.ingredients.map((ingredient, index) => (
                    <li key={ingredient.id || index} className="text-gray-700">
                      <span className="inline-flex items-center">
                        <span className="w-2 h-2 bg-blue-600 rounded-full mr-3 flex-shrink-0"></span>
                        {ingredient.original_text}
                      </span>
                      {ingredient.quantity && ingredient.unit && (
                        <span className="ml-2 text-sm text-gray-500">
                          ({ingredient.quantity} {ingredient.unit})
                        </span>
                      )}
                    </li>
                  ))}
                </ul>
              </section>
            )}

            {recipe.instructions && (
              <section className="mb-8">
                <h2 className="text-xl font-semibold text-gray-900 mb-4">Instructions</h2>
                <div className="prose max-w-none">
                  <div className="whitespace-pre-line text-gray-700 leading-relaxed">
                    {recipe.instructions}
                  </div>
                </div>
              </section>
            )}

            {recipe.tips && (
              <section className="mb-6">
                <h2 className="text-xl font-semibold text-gray-900 mb-4">Tips & Observations</h2>
                <div className="bg-yellow-50 border border-yellow-200 rounded-lg p-4">
                  <div className="whitespace-pre-line text-gray-700 leading-relaxed">
                    {recipe.tips}
                  </div>
                </div>
              </section>
            )}
          </div>

          <footer className="px-6 py-4 bg-gray-50 border-t border-gray-200 rounded-b-lg">
            <div className="flex items-center justify-between text-sm text-gray-600">
              <span>
                Created: {new Date(recipe.created_at).toLocaleDateString('en-US', {
                  month: 'long',
                  day: 'numeric',
                  year: 'numeric',
                  hour: 'numeric',
                  minute: '2-digit',
                })}
              </span>
              {recipe.updated_at !== recipe.created_at && (
                <span>
                  Updated: {new Date(recipe.updated_at).toLocaleDateString('en-US', {
                    month: 'long',
                    day: 'numeric',
                    year: 'numeric',
                    hour: 'numeric',
                    minute: '2-digit',
                  })}
                </span>
              )}
            </div>
          </footer>
        </article>
      </div>
    </div>
  );
}