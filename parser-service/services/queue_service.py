import redis
import json
import logging
import os
from typing import Dict, Any, Optional
from datetime import datetime

logger = logging.getLogger(__name__)

class QueueService:
    def __init__(self):
        redis_url = os.getenv('REDIS_URL', 'redis://localhost:6379')
        self.redis_client = redis.from_url(redis_url, decode_responses=True)
        self.processing_queue = 'recipe_processing'
        self.retry_queue = 'recipe_retry'

    async def enqueue_recipe_processing(self, recipe_id: str, image_urls: list[str]) -> bool:
        """Add a recipe processing job to the queue"""
        try:
            job_data = {
                'recipe_id': recipe_id,
                'image_urls': image_urls,
                'created_at': datetime.utcnow().isoformat(),
                'retry_count': 0
            }

            # Add to processing queue
            self.redis_client.lpush(self.processing_queue, json.dumps(job_data))
            logger.info(f"Enqueued recipe processing job for recipe {recipe_id}")
            return True

        except Exception as e:
            logger.error(f"Failed to enqueue recipe processing job: {e}")
            return False

    async def dequeue_recipe_processing(self) -> Optional[Dict[str, Any]]:
        """Get the next recipe processing job from the queue"""
        try:
            # Blocking pop from the right side (FIFO)
            result = self.redis_client.brpop(self.processing_queue, timeout=5)
            if result:
                _, job_json = result
                job_data = json.loads(job_json)
                logger.info(f"Dequeued recipe processing job for recipe {job_data['recipe_id']}")
                return job_data
            return None

        except Exception as e:
            logger.error(f"Failed to dequeue recipe processing job: {e}")
            return None

    async def requeue_with_retry(self, job_data: Dict[str, Any], error_message: str) -> bool:
        """Requeue a failed job with retry count increment"""
        try:
            retry_count = job_data.get('retry_count', 0) + 1
            max_retries = 3

            if retry_count <= max_retries:
                job_data['retry_count'] = retry_count
                job_data['last_error'] = error_message
                job_data['retried_at'] = datetime.utcnow().isoformat()

                # Add to retry queue with delay
                self.redis_client.lpush(self.retry_queue, json.dumps(job_data))
                logger.info(f"Requeued recipe {job_data['recipe_id']} for retry {retry_count}/{max_retries}")
                return True
            else:
                logger.error(f"Recipe {job_data['recipe_id']} exceeded max retries ({max_retries})")
                await self._move_to_failed_queue(job_data, error_message)
                return False

        except Exception as e:
            logger.error(f"Failed to requeue job: {e}")
            return False

    async def _move_to_failed_queue(self, job_data: Dict[str, Any], error_message: str):
        """Move job to failed queue for manual inspection"""
        try:
            job_data['failed_at'] = datetime.utcnow().isoformat()
            job_data['final_error'] = error_message
            self.redis_client.lpush('recipe_failed', json.dumps(job_data))
            logger.error(f"Moved recipe {job_data['recipe_id']} to failed queue")
        except Exception as e:
            logger.error(f"Failed to move job to failed queue: {e}")

    async def get_queue_stats(self) -> Dict[str, int]:
        """Get current queue statistics"""
        try:
            return {
                'processing': self.redis_client.llen(self.processing_queue),
                'retry': self.redis_client.llen(self.retry_queue),
                'failed': self.redis_client.llen('recipe_failed')
            }
        except Exception as e:
            logger.error(f"Failed to get queue stats: {e}")
            return {'processing': 0, 'retry': 0, 'failed': 0}