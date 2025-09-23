import logging
import requests
import json
from typing import Dict, Any, List, Optional
from .ocr_service import OCRService
from .llm_service import LLMService
from .queue_service import QueueService
import os

logger = logging.getLogger(__name__)

class ProcessingService:
    def __init__(self):
        """Initialize the processing service with all dependencies"""
        self.ocr_service = OCRService()
        self.llm_service = LLMService()
        self.queue_service = QueueService()
        self.api_service_url = os.getenv('API_SERVICE_URL', 'http://localhost:8080')

    async def process_recipe_images(self, job_data: Dict[str, Any]) -> bool:
        """Main processing pipeline: OCR -> LLM -> Database Update"""
        recipe_id = job_data['recipe_id']
        image_urls = job_data['image_urls']

        try:
            logger.info(f"Starting processing for recipe {recipe_id}")

            # Step 1: Update status to processing
            await self._update_recipe_status(recipe_id, "processing")

            # Step 2: Extract text from images using OCR
            logger.info(f"Extracting text from {len(image_urls)} images")
            extracted_texts = await self.ocr_service.extract_text_from_images(image_urls)

            if not extracted_texts:
                raise ValueError("No text could be extracted from any image")

            # Step 3: Structure the recipe using LLM
            logger.info("Structuring recipe data using LLM")
            structured_recipe = await self.llm_service.structure_recipe_from_text(extracted_texts)

            if not structured_recipe:
                raise ValueError("Failed to structure recipe data")

            # Step 4: Update recipe in database
            logger.info("Updating recipe in database")
            success = await self._update_recipe_data(recipe_id, structured_recipe)

            if success:
                # Step 5: Update status to review_required
                await self._update_recipe_status(recipe_id, "review_required")
                logger.info(f"Successfully processed recipe {recipe_id}")
                return True
            else:
                raise ValueError("Failed to update recipe in database")

        except Exception as e:
            error_msg = f"Processing failed for recipe {recipe_id}: {str(e)}"
            logger.error(error_msg)

            # Update recipe status to failed
            await self._update_recipe_status(recipe_id, "failed")

            # Requeue for retry if applicable
            await self.queue_service.requeue_with_retry(job_data, error_msg)
            return False

    async def _update_recipe_status(self, recipe_id: str, status: str) -> bool:
        """Update recipe status in the database"""
        try:
            url = f"{self.api_service_url}/api/v1/internal/recipes/{recipe_id}/status"
            payload = {"status": status}

            # Get internal service secret for authentication
            internal_secret = os.getenv('INTERNAL_SERVICE_SECRET')
            if not internal_secret:
                logger.error("INTERNAL_SERVICE_SECRET not configured")
                return False

            headers = {
                "Content-Type": "application/json",
                "X-Internal-Service-Auth": internal_secret
            }

            response = requests.put(
                url,
                json=payload,
                headers=headers,
                timeout=30
            )

            if response.status_code == 200:
                logger.info(f"Updated recipe {recipe_id} status to {status}")
                return True
            else:
                logger.error(f"Failed to update recipe status: {response.status_code} - {response.text}")
                return False

        except Exception as e:
            logger.error(f"Error updating recipe status: {e}")
            return False

    async def _update_recipe_data(self, recipe_id: str, recipe_data: Dict[str, Any]) -> bool:
        """Update recipe with structured data"""
        try:
            url = f"{self.api_service_url}/api/v1/internal/recipes/{recipe_id}"

            # Prepare the update payload
            update_payload = {
                "title": recipe_data.get("title"),
                "servings": recipe_data.get("servings"),
                "prep_time": recipe_data.get("prep_time"),
                "cook_time": recipe_data.get("cook_time"),
                "total_time": recipe_data.get("total_time"),
                "instructions": recipe_data.get("instructions", []),
                "tips": recipe_data.get("tips"),
                "notes": recipe_data.get("notes"),
                "ingredients": recipe_data.get("ingredients", [])
            }

            # Get internal service secret for authentication
            internal_secret = os.getenv('INTERNAL_SERVICE_SECRET')
            if not internal_secret:
                logger.error("INTERNAL_SERVICE_SECRET not configured")
                return False

            headers = {
                "Content-Type": "application/json",
                "X-Internal-Service-Auth": internal_secret
            }

            response = requests.put(
                url,
                json=update_payload,
                headers=headers,
                timeout=30
            )

            if response.status_code == 200:
                logger.info(f"Updated recipe {recipe_id} with structured data")
                return True
            else:
                logger.error(f"Failed to update recipe data: {response.status_code} - {response.text}")
                return False

        except Exception as e:
            logger.error(f"Error updating recipe data: {e}")
            return False

    async def process_single_recipe(self, recipe_id: str, image_urls: List[str]) -> Dict[str, Any]:
        """Process a single recipe immediately (for testing/debugging)"""
        try:
            job_data = {
                'recipe_id': recipe_id,
                'image_urls': image_urls,
                'retry_count': 0
            }

            success = await self.process_recipe_images(job_data)

            return {
                "success": success,
                "recipe_id": recipe_id,
                "message": "Processing completed" if success else "Processing failed"
            }

        except Exception as e:
            logger.error(f"Error in single recipe processing: {e}")
            return {
                "success": False,
                "recipe_id": recipe_id,
                "message": f"Processing error: {str(e)}"
            }

    async def get_processing_stats(self) -> Dict[str, Any]:
        """Get current processing statistics"""
        try:
            queue_stats = await self.queue_service.get_queue_stats()

            return {
                "queue_stats": queue_stats,
                "services": {
                    "ocr": "ready",
                    "llm": "ready",
                    "queue": "ready"
                }
            }

        except Exception as e:
            logger.error(f"Error getting processing stats: {e}")
            return {
                "queue_stats": {"processing": 0, "retry": 0, "failed": 0},
                "services": {
                    "ocr": "error",
                    "llm": "error",
                    "queue": "error"
                },
                "error": str(e)
            }