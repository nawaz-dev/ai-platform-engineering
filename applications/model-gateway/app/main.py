
from fastapi import FastAPI

from app.routers.chat import router as chat_router

app = FastAPI(
    title="FDE Enterprise Model Gateway",
    version="0.1.0",
)

app.include_router(chat_router)


@app.get("/health")
async def health():
    return {
        "status": "healthy",
        "service": "model-gateway",
    }
