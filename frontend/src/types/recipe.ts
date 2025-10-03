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
  pantry_item_id?: number;
  canonical_ingredient_id?: number; // Keep for backward compatibility during transition
  original_text: string;
  quantity?: number;
  unit?: string;
  suggested_pantry_item_name?: string; // AI-suggested name for new pantry items
  created_at: string;
  updated_at: string;
}

export interface PantryItem {
  id: number;
  name: string;
  is_approved: boolean;
  created_at: string;
  updated_at: string;
}

// Keep for backward compatibility during transition
export interface CanonicalIngredient {
  id: number;
  name: string;
  is_approved: boolean;
  created_at: string;
  updated_at: string;
}

export interface PantryItemManagement {
  id: number;
  name: string;
  is_approved: boolean;
  created_at: string;
  updated_at: string;
  usage_count: number;
}

// Keep for backward compatibility during transition
export interface IngredientManagement {
  id: number;
  name: string;
  is_approved: boolean;
  created_at: string;
  updated_at: string;
  usage_count: number;
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