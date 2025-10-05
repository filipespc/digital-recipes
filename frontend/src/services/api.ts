import axios from 'axios';
import { Recipe, RecipeWithIngredients, RecipeListResponse, StandardResponse, RecipeIngredient, CanonicalIngredient, PantryItem, PantryItemWithSimilarity, PantryItemManagement, IngredientManagement } from '@/types/recipe';
import { UPLOAD_CONFIG } from '@/constants/upload';

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080';

const apiClient = axios.create({
  baseURL: `${API_BASE_URL}/api/v1`,
  headers: {
    'Content-Type': 'application/json',
  },
});

export interface UploadURL {
  image_id: string;
  upload_url: string;
  fields: Record<string, string>;
}

export interface UploadRequestResponse {
  recipe_id: number;
  upload_urls: UploadURL[];
}

export class RecipeAPI {
  static async getRecipes(page = 1, perPage = 10, status?: string): Promise<RecipeListResponse> {
    const params = new URLSearchParams({
      page: page.toString(),
      per_page: perPage.toString(),
    });
    
    if (status) {
      params.append('status', status);
    }

    const response = await apiClient.get<StandardResponse<Recipe[]>>(
      `/recipes?${params.toString()}`
    );
    
    // Convert StandardResponse to RecipeListResponse format
    return {
      recipes: response.data.data,
      total: response.data.pagination?.total || 0,
      page: response.data.pagination?.page || 1,
      per_page: response.data.pagination?.per_page || perPage,
      total_pages: response.data.pagination?.total_pages || 0,
    };
  }

  static async getRecipe(id: number): Promise<RecipeWithIngredients> {
    const response = await apiClient.get<StandardResponse<RecipeWithIngredients>>(`/recipes/${id}`);
    return response.data.data;
  }

  static async updateRecipe(id: number, recipe: Partial<Recipe>): Promise<Recipe> {
    // Transform the recipe data to match backend expectations
    const updateData: any = {};

    if (recipe.title !== undefined) {
      updateData.title = recipe.title;
    }
    if (recipe.servings !== undefined) {
      updateData.servings = recipe.servings;
    }
    if (recipe.instructions !== undefined) {
      // Backend expects instructions as array of strings
      updateData.instructions = recipe.instructions ? recipe.instructions.split('\n').filter(line => line.trim()) : [];
    }
    if (recipe.tips !== undefined) {
      // Backend expects tips as array of strings
      updateData.tips = recipe.tips ? recipe.tips.split('\n').filter(line => line.trim()) : [];
    }

    const response = await apiClient.put<StandardResponse<Recipe>>(`/recipes/${id}`, updateData);
    return response.data.data;
  }

  static async updateRecipeStatus(id: number, status: string): Promise<{ recipe_id: number; status: string }> {
    const response = await apiClient.put<StandardResponse<{ recipe_id: number; status: string }>>(`/recipes/${id}/status`, { status });
    return response.data.data;
  }

  static async requestUpload(imageCount: number): Promise<UploadRequestResponse> {
    const response = await apiClient.post<StandardResponse<UploadRequestResponse>>(
      '/recipes/upload-request',
      { image_count: imageCount }
    );
    return response.data.data;
  }

  static async uploadImage(uploadUrl: UploadURL, file: File, onProgress?: (progress: number) => void): Promise<void> {
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      
      xhr.upload.addEventListener('progress', (event) => {
        if (event.lengthComputable && onProgress) {
          const progress = (event.loaded / event.total) * 100;
          onProgress(progress);
        }
      });

      xhr.addEventListener('load', () => {
        if (xhr.status >= 200 && xhr.status < 300) {
          resolve();
        } else {
          reject(new Error(`Upload failed with status: ${xhr.status}`));
        }
      });

      xhr.addEventListener('error', () => {
        reject(new Error('Upload failed'));
      });

      xhr.open('PUT', uploadUrl.upload_url);
      
      // Security: Validate and whitelist headers to prevent header injection
      const ALLOWED_HEADERS = UPLOAD_CONFIG.ALLOWED_UPLOAD_HEADERS;
      const METADATA_HEADER_PREFIX = UPLOAD_CONFIG.METADATA_HEADER_PREFIX;
      const MAX_HEADER_VALUE_LENGTH = UPLOAD_CONFIG.MAX_HEADER_VALUE_LENGTH;

      Object.entries(uploadUrl.fields).forEach(([key, value]) => {
        // Validate header name and value
        if (typeof key !== 'string' || typeof value !== 'string') {
          console.warn(`Invalid header type: ${key}=${value}`);
          return;
        }

        // Check if header is allowed
        const isAllowedHeader = ALLOWED_HEADERS.includes(key as typeof ALLOWED_HEADERS[number]) || 
                               key.startsWith(METADATA_HEADER_PREFIX);
        
        if (!isAllowedHeader) {
          console.warn(`Blocked potentially dangerous header: ${key}`);
          return;
        }

        // Validate header value
        if (value.length > MAX_HEADER_VALUE_LENGTH) {
          console.warn(`Header value too long: ${key}`);
          return;
        }

        // Prevent header injection attacks
        if (value.includes('\r') || value.includes('\n')) {
          console.warn(`Blocked header injection attempt: ${key}=${value}`);
          return;
        }

        try {
          xhr.setRequestHeader(key, value);
        } catch (error) {
          console.warn(`Failed to set header ${key}: ${error}`);
        }
      });
      
      xhr.send(file);
    });
  }

  // Ingredient Management API Methods

  static async addIngredient(recipeId: number, ingredient: {
    original_text: string;
    quantity?: number;
    unit?: string;
  }): Promise<RecipeIngredient> {
    const response = await apiClient.post<StandardResponse<RecipeIngredient>>(
      `/recipes/${recipeId}/ingredients`,
      ingredient
    );
    return response.data.data;
  }

  static async updateIngredient(
    recipeId: number,
    ingredientId: number,
    ingredient: {
      original_text?: string;
      quantity?: number;
      unit?: string;
    }
  ): Promise<RecipeIngredient> {
    const response = await apiClient.put<StandardResponse<RecipeIngredient>>(
      `/recipes/${recipeId}/ingredients/${ingredientId}`,
      ingredient
    );
    return response.data.data;
  }

  static async deleteIngredient(recipeId: number, ingredientId: number): Promise<void> {
    await apiClient.delete(`/recipes/${recipeId}/ingredients/${ingredientId}`);
  }

  // Primary Pantry Item API Methods
  static async searchPantryItems(query: string, userId: number): Promise<PantryItem[]> {
    const response = await apiClient.get<StandardResponse<PantryItem[]>>(
      `/pantry/search?q=${encodeURIComponent(query)}&user_id=${userId}`
    );
    return response.data.data;
  }

  static async fuzzySearchPantryItems(
    query: string,
    userId: number,
    threshold: number = 0.6
  ): Promise<PantryItemWithSimilarity[]> {
    const response = await apiClient.get<StandardResponse<PantryItemWithSimilarity[]>>(
      `/pantry/fuzzy-search?q=${encodeURIComponent(query)}&user_id=${userId}&threshold=${threshold}`
    );
    return response.data.data;
  }

  // Keep for backward compatibility during transition
  static async searchCanonicalIngredients(query: string, userId: number): Promise<CanonicalIngredient[]> {
    // Redirect to pantry search
    const response = await apiClient.get<StandardResponse<PantryItem[]>>(
      `/pantry/search?q=${encodeURIComponent(query)}&user_id=${userId}`
    );
    return response.data.data as any;
  }

  // Primary Pantry Item API Methods
  static async linkIngredientToPantryItem(
    recipeId: number,
    ingredientId: number,
    pantryItemId: number
  ): Promise<RecipeIngredient> {
    const response = await apiClient.put<StandardResponse<RecipeIngredient>>(
      `/recipes/${recipeId}/ingredients/${ingredientId}/link`,
      { pantry_item_id: pantryItemId }
    );
    return response.data.data;
  }

  static async createPantryItem(name: string, userId: number): Promise<PantryItem> {
    const response = await apiClient.post<StandardResponse<PantryItem>>(
      '/pantry',
      { name, user_id: userId }
    );
    return response.data.data;
  }

  // Keep for backward compatibility during transition
  static async linkIngredientToCanonical(
    recipeId: number,
    ingredientId: number,
    canonicalIngredientId: number
  ): Promise<RecipeIngredient> {
    const response = await apiClient.put<StandardResponse<RecipeIngredient>>(
      `/recipes/${recipeId}/ingredients/${ingredientId}/link`,
      { pantry_item_id: canonicalIngredientId }
    );
    return response.data.data;
  }

  static async createCanonicalIngredient(name: string, userId: number): Promise<CanonicalIngredient> {
    // Redirect to pantry
    const response = await apiClient.post<StandardResponse<PantryItem>>(
      '/pantry',
      { name, user_id: userId }
    );
    return response.data.data as any;
  }

  // Pantry Item Management API Methods

  static async getPantryItemManagement(userId: number): Promise<PantryItemManagement[]> {
    const response = await apiClient.get<StandardResponse<PantryItemManagement[]>>(
      `/pantry/manage?user_id=${userId}`
    );
    return response.data.data;
  }

  static async mergePantryItems(
    targetId: number,
    sourceId: number,
    userId: number
  ): Promise<{ message: string; target_id: number; target_name: string; source_name: string }> {
    const response = await apiClient.put<StandardResponse<any>>(
      `/pantry/${targetId}/merge`,
      { source_pantry_item_id: sourceId, user_id: userId }
    );
    return response.data.data;
  }

  static async updatePantryItem(
    pantryItemId: number,
    name: string,
    userId: number
  ): Promise<{ message: string; id: number; old_name: string; new_name: string }> {
    const response = await apiClient.put<StandardResponse<any>>(
      `/pantry/${pantryItemId}`,
      { name, user_id: userId }
    );
    return response.data.data;
  }

  static async deletePantryItem(pantryItemId: number, userId: number): Promise<{ message: string; id: number; name: string }> {
    const response = await apiClient.delete<StandardResponse<any>>(
      `/pantry/${pantryItemId}?user_id=${userId}`
    );
    return response.data.data;
  }

  // Keep for backward compatibility during transition
  static async getIngredientManagement(userId: number): Promise<IngredientManagement[]> {
    // Redirect to pantry
    const response = await apiClient.get<StandardResponse<PantryItemManagement[]>>(
      `/pantry/manage?user_id=${userId}`
    );
    return response.data.data as any;
  }

  static async mergeCanonicalIngredients(
    targetId: number,
    sourceId: number,
    userId: number
  ): Promise<{ message: string; target_id: number; target_name: string; source_name: string }> {
    // Redirect to pantry
    const response = await apiClient.put<StandardResponse<any>>(
      `/pantry/${targetId}/merge`,
      { source_pantry_item_id: sourceId, user_id: userId }
    );
    return response.data.data;
  }

  static async updateCanonicalIngredient(
    ingredientId: number,
    name: string,
    userId: number
  ): Promise<{ message: string; id: number; old_name: string; new_name: string }> {
    // Redirect to pantry
    const response = await apiClient.put<StandardResponse<any>>(
      `/pantry/${ingredientId}`,
      { name, user_id: userId }
    );
    return response.data.data;
  }

  static async deleteCanonicalIngredient(ingredientId: number, userId: number): Promise<{ message: string; id: number; name: string }> {
    // Redirect to pantry
    const response = await apiClient.delete<StandardResponse<any>>(
      `/pantry/${ingredientId}?user_id=${userId}`
    );
    return response.data.data;
  }

  // AI-powered pantry name suggestions
  static async suggestPantryItemName(
    ingredientText: string,
    existingPantryItems?: string[]
  ): Promise<{
    suggested_name: string;
    confidence: number;
    reasoning: string;
    original_text: string;
  }> {
    const response = await apiClient.post<StandardResponse<{
      suggested_name: string;
      confidence: number;
      reasoning: string;
      original_text: string;
    }>>('/pantry/suggest-name', {
      ingredient_text: ingredientText,
      existing_pantry_items: existingPantryItems || []
    });

    return response.data.data;
  }
}

export default apiClient;