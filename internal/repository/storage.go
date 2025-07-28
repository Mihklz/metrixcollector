package repository

type Storage interface {
	Update(metricType, name, value string) error
}
