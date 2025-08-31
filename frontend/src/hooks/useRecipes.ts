import { useState, useEffect } from 'react';
import { Recipe, RecipeWithIngredients, RecipeListResponse } from '@/types/recipe';
import { RecipeAPI } from '@/services/api';

export function useRecipes(page = 1, perPage = 10, status?: string) {
  const [data, setData] = useState<RecipeListResponse | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchRecipes = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await RecipeAPI.getRecipes(page, perPage, status);
        setData(response);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch recipes');
      } finally {
        setLoading(false);
      }
    };

    fetchRecipes();
  }, [page, perPage, status]);

  return { data, loading, error, refetch: () => setData(null) };
}

export function useRecipe(id: number) {
  const [recipe, setRecipe] = useState<RecipeWithIngredients | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const fetchRecipe = async () => {
      try {
        setLoading(true);
        setError(null);
        const response = await RecipeAPI.getRecipe(id);
        setRecipe(response);
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to fetch recipe');
      } finally {
        setLoading(false);
      }
    };

    if (id) {
      fetchRecipe();
    }
  }, [id]);

  return { recipe, loading, error };
}