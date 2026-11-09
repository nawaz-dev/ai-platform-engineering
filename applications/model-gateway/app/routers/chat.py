
import httpx
from fastapi import APIRouter, HTTPException, Request
from fastapi.responses import JSONResponse

from app.config import settings
from app.services.vllm_client import chat_completion

router = APIRouter(prefix="/v1", tags=["chat"])


@router.post("/chat/completions")
async def create_chat_completion(request: Request):
    try:
        payload = await request.json()

        if not isinstance(payload, dict):
            raise HTTPException(
                status_code=400,
                detail="Expected a JSON object",
            )

        payload.setdefault("model", settings.default_model)

        result = await chat_completion(payload)
        return JSONResponse(content=result)

    except httpx.TimeoutException:
        raise HTTPException(
            status_code=504,
            detail="Upstream vLLM request timed out",
        )

    except httpx.HTTPStatusError as exc:
        raise HTTPException(
            status_code=exc.response.status_code,
            detail=exc.response.text,
        )

    except httpx.RequestError:
        raise HTTPException(
            status_code=502,
            detail="Unable to connect to vLLM",
        )
