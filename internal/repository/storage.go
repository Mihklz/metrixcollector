package repository

type Storage interface {
	Update(metricType, name, value string) error
	GetGauge(name string) (Gauge, bool)
	GetCounter(name string) (Counter, bool)
	GetAllGauges() map[string]Gauge
	GetAllCounters() map[string]Counter
}
