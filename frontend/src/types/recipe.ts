export interface Recipe {
  id: number;
  title: string;
  servings?: string;
  instructions?: string;
  tips?: string;
  status: string;
  user_id: number;
  created_at: string;
  updated_at: string;
}

export interface RecipeIngredient {
  id: number;
  recipe_id: number;
  canonical_ingredient_id?: number;
  original_text: string;
  quantity?: number;
  unit?: string;
  created_at: string;
  updated_at: string;
}

export interface RecipeWithIngredients extends Recipe {
  ingredients?: RecipeIngredient[];
}

export interface ApiResponse<T> {
  data: T;
  message?: string;
}

export interface RecipeListResponse {
  recipes: Recipe[];
  total: number;
  page: number;
  per_page: number;
  total_pages: number;
}

export interface StandardResponse<T> {
  data: T;
  pagination?: {
    page: number;
    per_page: number;
    total: number;
    total_pages: number;
  };
  meta?: {
    request_id?: string;
    timestamp?: string;
  };
  error?: string;
}