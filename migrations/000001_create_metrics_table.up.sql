-- Создание таблицы метрик
CREATE TABLE IF NOT EXISTS metrics (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('gauge', 'counter')),
    delta BIGINT,
    value DOUBLE PRECISION,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ограничение: delta используется только для counter, value только для gauge
    CONSTRAINT check_metric_values CHECK (
        (type = 'counter' AND delta IS NOT NULL AND value IS NULL) OR
        (type = 'gauge' AND value IS NOT NULL AND delta IS NULL)
    )
);

-- Уникальный индекс для комбинации name + type (одно имя метрики может быть только одного типа)
CREATE UNIQUE INDEX IF NOT EXISTS idx_metrics_name_type ON metrics(name, type);

-- Индекс для быстрого поиска по типу
CREATE INDEX IF NOT EXISTS idx_metrics_type ON metrics(type);

-- Индекс для сортировки по времени обновления
CREATE INDEX IF NOT EXISTS idx_metrics_updated_at ON metrics(updated_at);
