-- Откат создания таблицы метрик
DROP INDEX IF EXISTS idx_metrics_updated_at;
DROP INDEX IF EXISTS idx_metrics_type;
DROP INDEX IF EXISTS idx_metrics_name_type;
DROP TABLE IF EXISTS metrics;
