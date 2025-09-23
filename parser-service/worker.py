import asyncio
import logging
import signal
import sys
import os
from services.processing_service import ProcessingService
from services.queue_service import QueueService

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

class RecipeWorker:
    def __init__(self):
        self.processing_service = ProcessingService()
        self.queue_service = QueueService()
        self.running = True

    async def start_processing(self):
        """Main worker loop to process recipe jobs"""
        logger.info("Recipe processing worker started")

        while self.running:
            try:
                # Get next job from queue
                job = await self.queue_service.dequeue_recipe_processing()

                if job:
                    # Process the recipe
                    await self.processing_service.process_recipe_images(job)
                else:
                    # No jobs available, wait a bit
                    await asyncio.sleep(1)

            except KeyboardInterrupt:
                logger.info("Received interrupt signal, shutting down...")
                self.running = False
                break
            except Exception as e:
                logger.error(f"Unexpected error in worker loop: {e}")
                # Continue processing other jobs
                await asyncio.sleep(5)

        logger.info("Recipe processing worker stopped")

    def stop(self):
        """Stop the worker gracefully"""
        self.running = False

async def main():
    worker = RecipeWorker()

    # Handle shutdown signals
    def signal_handler(signum, frame):
        logger.info(f"Received signal {signum}, shutting down worker...")
        worker.stop()

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    try:
        await worker.start_processing()
    except Exception as e:
        logger.error(f"Worker failed: {e}")
        sys.exit(1)

if __name__ == "__main__":
    asyncio.run(main())