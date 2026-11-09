
import httpx

from app.config import settings


async def chat_completion(payload: dict) -> dict:
    async with httpx.AsyncClient(
        base_url=settings.vllm_base_url,
        timeout=settings.request_timeout,
    ) as client:
        response = await client.post(
            "/v1/chat/completions",
            json=payload,
        )
        response.raise_for_status()
        return response.json()
