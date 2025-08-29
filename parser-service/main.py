from fastapi import FastAPI
from pydantic import BaseModel

app = FastAPI(title="Digital Recipes Parser Service", version="1.0.0")

class HealthResponse(BaseModel):
    status: str
    service: str

@app.get("/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(
        status="healthy",
        service="digital-recipes-parser"
    )

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8081)