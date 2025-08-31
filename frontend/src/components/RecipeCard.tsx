import Link from 'next/link';
import { Recipe } from '@/types/recipe';

interface RecipeCardProps {
  recipe: Recipe;
}

export default function RecipeCard({ recipe }: RecipeCardProps) {
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

  const formatDate = (dateString: string) => {
    return new Date(dateString).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  };

  return (
    <div className="bg-white rounded-lg shadow-sm border border-gray-200 hover:shadow-md transition-shadow">
      <div className="p-6">
        <div className="flex items-start justify-between mb-3">
          <Link
            href={`/recipes/${recipe.id}`}
            className="text-lg font-semibold text-gray-900 hover:text-blue-600 transition-colors"
          >
            {recipe.title}
          </Link>
          <span
            className={`inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium ${getStatusColor(
              recipe.status
            )}`}
          >
            {recipe.status.replace('_', ' ')}
          </span>
        </div>
        
        <div className="text-sm text-gray-600 space-y-1">
          {recipe.servings && (
            <p>
              <span className="font-medium">Servings:</span> {recipe.servings}
            </p>
          )}
          <p>
            <span className="font-medium">Created:</span> {formatDate(recipe.created_at)}
          </p>
        </div>

        <div className="mt-4 pt-3 border-t border-gray-100">
          <Link
            href={`/recipes/${recipe.id}`}
            className="text-blue-600 hover:text-blue-800 text-sm font-medium"
          >
            View Recipe →
          </Link>
        </div>
      </div>
    </div>
  );
}