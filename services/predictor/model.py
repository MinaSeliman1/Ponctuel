"""Reproducible, tabular prediction-error model.

The target is the observed arrival error in seconds.  A prediction of zero is
the comparable STM-style baseline: it keeps the existing arrival estimate and
does not add a machine-learning correction.
"""

from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime
from math import ceil
from pathlib import Path
from typing import Any, Sequence

import joblib
import numpy as np
from sklearn.ensemble import GradientBoostingRegressor


@dataclass(frozen=True)
class Observation:
    """One labelled arrival-error observation."""

    route_id: str
    horizon_seconds: int
    hour: int
    delay_seconds: float
    error_seconds: float
    recorded_at: datetime


@dataclass(frozen=True)
class Metrics:
    """A metric together with the number of samples behind it."""

    sample_count: int
    mae_seconds: float


@dataclass(frozen=True)
class TrainingResult:
    """Model and evaluation data produced by a temporal training run."""

    model: GradientBoostingRegressor
    route_codes: dict[str, int]
    baseline: Metrics
    model_metrics: Metrics
    train_count: int
    test_count: int
    train_end: datetime
    test_start: datetime


def temporal_split(
    observations: Sequence[Observation], test_fraction: float = 0.2
) -> tuple[list[Observation], list[Observation]]:
    """Split observations chronologically, keeping the newest rows for test."""

    if not 0 < test_fraction < 1:
        raise ValueError("test_fraction must be between 0 and 1")
    ordered = sorted(observations, key=lambda observation: observation.recorded_at)
    if len(ordered) < 2:
        raise ValueError("at least two observations are required")
    test_count = max(1, ceil(len(ordered) * test_fraction))
    train_count = len(ordered) - test_count
    if train_count < 1:
        raise ValueError("the temporal split must leave at least one training row")
    return ordered[:train_count], ordered[train_count:]


def route_mapping(observations: Sequence[Observation]) -> dict[str, int]:
    """Build a stable route encoding without learning from row order."""

    return {route_id: index for index, route_id in enumerate(sorted({o.route_id for o in observations}))}


def feature_matrix(
    observations: Sequence[Observation], route_codes: dict[str, int] | None = None
) -> np.ndarray:
    """Convert observations to the model's fixed four-column feature contract."""

    codes = route_codes if route_codes is not None else route_mapping(observations)
    return np.asarray(
        [
            [
                float(observation.horizon_seconds),
                float(observation.hour),
                float(observation.delay_seconds),
                float(codes.get(observation.route_id, -1)),
            ]
            for observation in observations
        ],
        dtype=float,
    ).reshape((-1, 4))


def baseline_metrics(observations: Sequence[Observation]) -> Metrics:
    """Evaluate the no-correction baseline (predicted error is always zero)."""

    errors = np.asarray([observation.error_seconds for observation in observations], dtype=float)
    if errors.size == 0:
        raise ValueError("at least one observation is required")
    return Metrics(sample_count=int(errors.size), mae_seconds=float(np.mean(np.abs(errors))))


def _mae(observations: Sequence[Observation], predictions: np.ndarray) -> Metrics:
    actual = np.asarray([observation.error_seconds for observation in observations], dtype=float)
    return Metrics(sample_count=len(observations), mae_seconds=float(np.mean(np.abs(actual - predictions))))


def train_gradient_boosting(
    observations: Sequence[Observation], test_fraction: float = 0.2
) -> TrainingResult:
    """Train and evaluate with a chronological holdout.

    Route codes are learned from the training partition only.  This prevents
    a future-only route from influencing the training feature contract.
    """

    train, test = temporal_split(observations, test_fraction)
    route_codes = route_mapping(train)
    model = GradientBoostingRegressor(
        learning_rate=0.05,
        loss="huber",
        max_depth=2,
        n_estimators=50,
        random_state=42,
    )
    model.fit(feature_matrix(train, route_codes), [o.error_seconds for o in train])
    predictions = model.predict(feature_matrix(test, route_codes))
    return TrainingResult(
        model=model,
        route_codes=route_codes,
        baseline=baseline_metrics(test),
        model_metrics=_mae(test, predictions),
        train_count=len(train),
        test_count=len(test),
        train_end=train[-1].recorded_at,
        test_start=test[0].recorded_at,
    )


def save_artifact(result: TrainingResult, path: str | Path) -> None:
    """Persist only the trained estimator and its feature encoding."""

    destination = Path(path)
    destination.parent.mkdir(parents=True, exist_ok=True)
    joblib.dump(
        {
            "model": result.model,
            "route_codes": result.route_codes,
            "metadata": {
                "baseline": result.baseline.__dict__,
                "model": result.model_metrics.__dict__,
                "train_count": result.train_count,
                "test_count": result.test_count,
                "train_end": result.train_end.isoformat(),
                "test_start": result.test_start.isoformat(),
            },
        },
        destination,
    )


def load_artifact(path: str | Path) -> dict[str, Any]:
    """Load and minimally validate a model artifact created by this module."""

    artifact = joblib.load(path)
    if not isinstance(artifact, dict) or not {"model", "route_codes"}.issubset(artifact):
        raise ValueError("invalid predictor artifact")
    return artifact
