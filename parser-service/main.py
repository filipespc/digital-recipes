from fastapi import FastAPI, BackgroundTasks, HTTPException
from pydantic import BaseModel
from typing import List, Dict, Any
import logging
import asyncio
import threading
from services.processing_service import ProcessingService
from services.queue_service import QueueService
from worker import RecipeWorker

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)

app = FastAPI(title="Digital Recipes Parser Service", version="1.0.0")

# Initialize services
processing_service = ProcessingService()
queue_service = QueueService()

# Global worker instance
worker_instance = None

class HealthResponse(BaseModel):
    status: str
    service: str

class ProcessingRequest(BaseModel):
    recipe_id: str
    image_urls: List[str]

class ProcessingResponse(BaseModel):
    success: bool
    message: str
    recipe_id: str

class QueueStatsResponse(BaseModel):
    queue_stats: Dict[str, int]
    services: Dict[str, str]

@app.get("/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(
        status="healthy",
        service="digital-recipes-parser"
    )

@app.post("/process", response_model=ProcessingResponse)
async def enqueue_processing(request: ProcessingRequest):
    """Enqueue a recipe for processing"""
    try:
        success = await queue_service.enqueue_recipe_processing(
            request.recipe_id,
            request.image_urls
        )

        if success:
            return ProcessingResponse(
                success=True,
                message="Recipe enqueued for processing",
                recipe_id=request.recipe_id
            )
        else:
            raise HTTPException(status_code=500, detail="Failed to enqueue recipe")

    except Exception as e:
        logger.error(f"Error enqueuing recipe {request.recipe_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/process-immediate", response_model=ProcessingResponse)
async def process_immediate(request: ProcessingRequest):
    """Process a recipe immediately (for testing)"""
    try:
        result = await processing_service.process_single_recipe(
            request.recipe_id,
            request.image_urls
        )

        return ProcessingResponse(
            success=result["success"],
            message=result["message"],
            recipe_id=result["recipe_id"]
        )

    except Exception as e:
        logger.error(f"Error processing recipe {request.recipe_id}: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/stats", response_model=QueueStatsResponse)
async def get_stats():
    """Get processing statistics"""
    try:
        stats = await processing_service.get_processing_stats()
        return QueueStatsResponse(
            queue_stats=stats["queue_stats"],
            services=stats["services"]
        )
    except Exception as e:
        logger.error(f"Error getting stats: {e}")
        raise HTTPException(status_code=500, detail=str(e))

@app.on_event("startup")
async def startup_event():
    """Start the background worker on application startup"""
    global worker_instance
    try:
        logger.info("Starting background recipe processing worker...")
        worker_instance = RecipeWorker()

        # Start worker in background thread
        def run_worker():
            loop = asyncio.new_event_loop()
            asyncio.set_event_loop(loop)
            loop.run_until_complete(worker_instance.start_processing())

        worker_thread = threading.Thread(target=run_worker, daemon=True)
        worker_thread.start()

        logger.info("Background worker started successfully")
    except Exception as e:
        logger.error(f"Failed to start background worker: {e}")

@app.on_event("shutdown")
async def shutdown_event():
    """Stop the background worker on application shutdown"""
    global worker_instance
    if worker_instance:
        logger.info("Stopping background worker...")
        worker_instance.stop()
        logger.info("Background worker stopped")

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8081)