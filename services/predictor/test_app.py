import services.predictor.app as predictor_app
from fastapi.testclient import TestClient


class FakeModel:
    def predict(self, features):
        assert features.shape == (1, 4)
        return [12.5]


def test_health_and_predict_contract(monkeypatch) -> None:
    monkeypatch.setattr(
        predictor_app,
        "MODEL_ARTIFACT",
        {"model": FakeModel(), "route_codes": {"51": 0}},
    )
    client = TestClient(predictor_app.app)

    assert client.get("/healthz").json() == {"status": "ok"}
    response = client.post(
        "/predict",
        json={"route_id": "51", "horizon_seconds": 300, "hour": 7, "delay_seconds": 60},
    )
    assert response.status_code == 200
    assert response.json() == {"predictedErrorSeconds": 12.5}


def test_predict_rejects_invalid_horizon(monkeypatch) -> None:
    monkeypatch.setattr(
        predictor_app,
        "MODEL_ARTIFACT",
        {"model": FakeModel(), "route_codes": {"51": 0}},
    )
    response = TestClient(predictor_app.app).post(
        "/predict",
        json={"route_id": "51", "horizon_seconds": 3601, "hour": 7, "delay_seconds": 0},
    )

    assert response.status_code == 422


def test_predict_returns_503_without_artifact(monkeypatch) -> None:
    monkeypatch.setattr(predictor_app, "MODEL_ARTIFACT", None)
    response = TestClient(predictor_app.app).post(
        "/predict",
        json={"route_id": "51", "horizon_seconds": 300, "hour": 7, "delay_seconds": 0},
    )

    assert response.status_code == 503
