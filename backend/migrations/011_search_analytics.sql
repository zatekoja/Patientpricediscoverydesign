CREATE TABLE IF NOT EXISTS search_analytics (
    id UUID PRIMARY KEY,
    query TEXT NOT NULL,
    normalized_query TEXT,
    detected_intent VARCHAR(50),
    intent_confidence FLOAT,
    result_count INT NOT NULL DEFAULT 0,
    latency_ms INT,
    user_latitude FLOAT,
    user_longitude FLOAT,
    session_id VARCHAR(255),
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_analytics_query ON search_analytics (query);
CREATE INDEX IF NOT EXISTS idx_search_analytics_result_count ON search_analytics (result_count);
CREATE INDEX IF NOT EXISTS idx_search_analytics_created_at ON search_analytics (created_at);
