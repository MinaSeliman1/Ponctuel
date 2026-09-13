"""FastAPI contract for a trained Ponctuel prediction-error model."""

from __future__ import annotations

import os
from datetime import datetime, timezone
from pathlib import Path
from typing import Any

from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, ConfigDict, Field

from .model import Observation, feature_matrix, load_artifact


class PredictionRequest(BaseModel):
    route_id: str = Field(min_length=1, max_length=64)
    horizon_seconds: int = Field(ge=0, le=3600)
    hour: int = Field(ge=0, le=23)
    delay_seconds: float


class PredictionResponse(BaseModel):
    model_config = ConfigDict(populate_by_name=True)

    predicted_error_seconds: float = Field(alias="predictedErrorSeconds")


def _load_configured_artifact() -> dict[str, Any] | None:
    configured_path = os.getenv("PREDICTOR_MODEL_PATH", "")
    if not configured_path:
        return None
    path = Path(configured_path)
    if not path.is_file():
        return None
    try:
        return load_artifact(path)
    except (OSError, ValueError, TypeError):
        return None


MODEL_ARTIFACT = _load_configured_artifact()
app = FastAPI(title="Ponctuel predictor", version="0.4.0")


@app.get("/healthz")
def healthz() -> dict[str, str]:
    """Liveness endpoint; model availability is reported by /predict."""

    return {"status": "ok" if MODEL_ARTIFACT is not None else "waiting_for_model"}


@app.post("/predict", response_model=PredictionResponse)
def predict(request: PredictionRequest) -> PredictionResponse:
    if MODEL_ARTIFACT is None:
        raise HTTPException(status_code=503, detail="trained model artifact unavailable")
    observation = Observation(
        route_id=request.route_id,
        horizon_seconds=request.horizon_seconds,
        hour=request.hour,
        delay_seconds=request.delay_seconds,
        error_seconds=0.0,
        recorded_at=datetime.now(timezone.utc),
    )
    features = feature_matrix([observation], MODEL_ARTIFACT["route_codes"])
    predicted = float(MODEL_ARTIFACT["model"].predict(features)[0])
    return PredictionResponse(predictedErrorSeconds=predicted)
