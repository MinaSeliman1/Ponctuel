CREATE INDEX IF NOT EXISTS prediction_error_arrival_idx
    ON prediction_error (arrival_id, calculated_at DESC);

CREATE INDEX IF NOT EXISTS prediction_error_prediction_idx
    ON prediction_error (prediction_id, prediction_recorded_at);
