import axios from 'axios';
import { Recipe, RecipeWithIngredients, RecipeListResponse, StandardResponse } from '@/types/recipe';
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
    const response = await apiClient.put<Recipe>(`/recipes/${id}`, recipe);
    return response.data;
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
}

export default apiClient;