from datetime import datetime, timedelta, timezone

import pytest

from .model import (
    Observation,
    baseline_metrics,
    route_mapping,
    temporal_split,
    train_gradient_boosting,
)


def observations(count: int = 10) -> list[Observation]:
    start = datetime(2026, 1, 1, tzinfo=timezone.utc)
    return [
        Observation(
            route_id="80" if index % 2 else "51",
            horizon_seconds=300 + index * 30,
            hour=7 + index % 4,
            delay_seconds=float(index * 5),
            error_seconds=float(index - 3),
            recorded_at=start + timedelta(minutes=index),
        )
        for index in range(count)
    ]


def test_temporal_split_sorts_and_reserves_the_future() -> None:
    rows = list(reversed(observations(5)))
    train, test = temporal_split(rows, test_fraction=0.4)

    assert [row.recorded_at for row in train] == sorted(row.recorded_at for row in train)
    assert max(row.recorded_at for row in train) < min(row.recorded_at for row in test)
    assert len(train) == 3
    assert len(test) == 2


def test_baseline_predicts_no_additional_error() -> None:
    metrics = baseline_metrics(observations(4))

    assert metrics.sample_count == 4
    assert metrics.mae_seconds == pytest.approx(1.5)


def test_route_mapping_is_deterministic() -> None:
    assert route_mapping(observations(4)) == {"51": 0, "80": 1}


def test_training_uses_only_training_routes_and_reports_comparable_metrics() -> None:
    rows = observations(12)
    rows[-1] = Observation(
        route_id="future-only-route",
        horizon_seconds=900,
        hour=12,
        delay_seconds=0,
        error_seconds=3,
        recorded_at=rows[-1].recorded_at,
    )
    result = train_gradient_boosting(rows, test_fraction=0.25)

    assert "future-only-route" not in result.route_codes
    assert result.train_end < result.test_start
    assert result.baseline.sample_count == result.test_count
    assert result.model_metrics.sample_count == result.test_count
