import logging
import os
import json
from typing import Dict, Any, List, Optional
import google.generativeai as genai
from pydantic import BaseModel, ValidationError

logger = logging.getLogger(__name__)

class RecipeIngredient(BaseModel):
    quantity: Optional[str] = None
    unit: Optional[str] = None
    name: str
    notes: Optional[str] = None

class StructuredRecipe(BaseModel):
    title: str
    servings: Optional[str] = None
    prep_time: Optional[str] = None
    cook_time: Optional[str] = None
    total_time: Optional[str] = None
    ingredients: List[RecipeIngredient]
    instructions: List[str]
    tips: Optional[List[str]] = None
    notes: Optional[str] = None

class LLMService:
    def __init__(self):
        """Initialize Google Gemini API client"""
        try:
            api_key = os.getenv('GEMINI_API_KEY')
            if not api_key:
                raise ValueError("GEMINI_API_KEY environment variable not set")

            genai.configure(api_key=api_key)
            self.model = genai.GenerativeModel('gemini-2.0-flash-exp')

            # Configuration from environment
            self.max_tokens = int(os.getenv('LLM_MAX_TOKENS', '2000'))
            self.temperature = float(os.getenv('LLM_TEMPERATURE', '0.1'))

            logger.info("Google Gemini API client initialized successfully")
        except Exception as e:
            logger.error(f"Failed to initialize Gemini API client: {e}")
            raise

    async def structure_recipe_from_text(self, extracted_texts: List[str]) -> Optional[Dict[str, Any]]:
        """Convert extracted OCR text into structured recipe data"""
        try:
            # Combine all extracted texts
            combined_text = "\n\n".join(extracted_texts)

            if not combined_text.strip():
                raise ValueError("No text provided for processing")

            # Create the prompt for recipe structuring
            prompt = self._create_recipe_structuring_prompt(combined_text)

            # Generate structured recipe using Gemini
            response = self.model.generate_content(
                prompt,
                generation_config=genai.types.GenerationConfig(
                    max_output_tokens=self.max_tokens,
                    temperature=self.temperature,
                )
            )

            if not response.text:
                raise ValueError("Empty response from Gemini API")

            # Parse and validate the JSON response
            structured_recipe = self._parse_and_validate_response(response.text)

            logger.info("Successfully structured recipe from text")
            return structured_recipe

        except Exception as e:
            logger.error(f"Failed to structure recipe from text: {e}")
            raise

    def _create_recipe_structuring_prompt(self, text: str) -> str:
        """Create a detailed prompt for recipe structuring"""
        return f"""
You are a professional recipe parser. Extract and structure the following recipe text into a valid JSON format.

IMPORTANT INSTRUCTIONS:
1. Return ONLY valid JSON, no additional text or formatting
2. Follow the exact schema provided below
3. Be conservative - if information is unclear or missing, use null
4. For ingredients, extract quantity, unit, and name separately
5. Break instructions into clear, numbered steps
6. Extract any cooking tips or notes if present

JSON SCHEMA TO FOLLOW:
{{
    "title": "string (recipe name)",
    "servings": "string or null (e.g., '4 servings', '6-8 people')",
    "prep_time": "string or null (e.g., '15 minutes')",
    "cook_time": "string or null (e.g., '30 minutes')",
    "total_time": "string or null (e.g., '45 minutes')",
    "ingredients": [
        {{
            "quantity": "string or null (e.g., '2', '1/2')",
            "unit": "string or null (e.g., 'cups', 'tbsp', 'lbs')",
            "name": "string (ingredient name)",
            "notes": "string or null (preparation notes like 'chopped', 'diced')"
        }}
    ],
    "instructions": [
        "string (step 1)",
        "string (step 2)",
        "string (step 3)"
    ],
    "tips": ["string or null (cooking tips)"],
    "notes": "string or null (additional notes)"
}}

RECIPE TEXT TO PARSE:
{text}

Return the structured recipe as valid JSON:"""

    def _parse_and_validate_response(self, response_text: str) -> Dict[str, Any]:
        """Parse and validate the LLM response"""
        try:
            # Clean up the response text
            cleaned_text = response_text.strip()

            # Remove any markdown code blocks if present
            if cleaned_text.startswith('```json'):
                cleaned_text = cleaned_text[7:]
            if cleaned_text.startswith('```'):
                cleaned_text = cleaned_text[3:]
            if cleaned_text.endswith('```'):
                cleaned_text = cleaned_text[:-3]

            cleaned_text = cleaned_text.strip()

            # Parse JSON
            recipe_data = json.loads(cleaned_text)

            # Validate using Pydantic model
            validated_recipe = StructuredRecipe(**recipe_data)

            # Convert back to dict for database storage
            return validated_recipe.model_dump()

        except json.JSONDecodeError as e:
            logger.error(f"Invalid JSON in LLM response: {e}")
            logger.error(f"Response text: {response_text}")
            raise ValueError(f"LLM returned invalid JSON: {e}")

        except ValidationError as e:
            logger.error(f"Recipe validation failed: {e}")
            logger.error(f"Recipe data: {recipe_data}")
            raise ValueError(f"Recipe structure validation failed: {e}")

        except Exception as e:
            logger.error(f"Unexpected error parsing LLM response: {e}")
            raise

    async def enhance_ingredient_extraction(self, ingredient_text: str) -> Dict[str, Any]:
        """Enhanced ingredient parsing for complex ingredient strings"""
        try:
            prompt = f"""
Parse this ingredient text into structured format. Return ONLY valid JSON.

Ingredient text: "{ingredient_text}"

Return JSON format:
{{
    "quantity": "string or null",
    "unit": "string or null",
    "name": "string",
    "notes": "string or null"
}}
"""

            response = self.model.generate_content(
                prompt,
                generation_config=genai.types.GenerationConfig(
                    max_output_tokens=200,
                    temperature=0.1,
                )
            )

            if not response.text:
                raise ValueError("Empty response for ingredient parsing")

            # Parse and return ingredient data
            ingredient_data = json.loads(response.text.strip())
            return ingredient_data

        except Exception as e:
            logger.error(f"Failed to enhance ingredient extraction: {e}")
            # Return basic parsing as fallback
            return {
                "quantity": None,
                "unit": None,
                "name": ingredient_text,
                "notes": None
            }