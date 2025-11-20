# go-musthave-metrics-tpl

Шаблон репозитория для трека «Сервер сбора метрик и алертинга».

## Начало работы

1. Склонируйте репозиторий в любую подходящую директорию на вашем компьютере.
2. В корне репозитория выполните команду `go mod init <name>` (где `<name>` — адрес вашего репозитория на GitHub без префикса `https://`) для создания модуля.

## Обновление шаблона

Чтобы иметь возможность получать обновления автотестов и других частей шаблона, выполните команду:

```
git remote add -m main template https://github.com/Yandex-Practicum/go-musthave-metrics-tpl.git
```

Для обновления кода автотестов выполните команду:

```
git fetch template && git checkout template/main .github
```

Затем добавьте полученные изменения в свой репозиторий.

## Запуск автотестов

Для успешного запуска автотестов называйте ветки `iter<number>`, где `<number>` — порядковый номер инкремента. Например, в ветке с названием `iter4` запустятся автотесты для инкрементов с первого по четвёртый.

При мёрже ветки с инкрементом в основную ветку `main` будут запускаться все автотесты.

Подробнее про локальный и автоматический запуск читайте в [README автотестов](https://github.com/Yandex-Practicum/go-autotests).

## Структура проекта

Приведённая в этом репозитории структура проекта является рекомендуемой, но не обязательной.

Это лишь пример организации кода, который поможет вам в реализации сервиса.

При необходимости можно вносить изменения в структуру проекта, использовать любые библиотеки и предпочитаемые структурные паттерны организации кода приложения, например:
- **DDD** (Domain-Driven Design)
- **Clean Architecture**
- **Hexagonal Architecture**
- **Layered Architecture**

## Оптимизация производительности

### Memory profile diff

Результаты сравнения профилей памяти до и после оптимизации `MemStorage.SaveToFile`:

```bash
$ go test -bench=BenchmarkMemStorageSaveToFile -memprofile=profiles/base.pprof ./internal/repository
$ # ... внесли оптимизацию ...
$ go test -bench=BenchmarkMemStorageSaveToFile -memprofile=profiles/result.pprof ./internal/repository
$ go tool pprof -top -diff_base=profiles/base.pprof profiles/result.pprof
```

Результат:
```
File: repository.test
Type: alloc_space
Showing nodes accounting for -958.54MB, 54.90% of 1746.10MB total
      flat  flat%   sum%        cum   cum%
 -707.17MB 40.50% 40.50%  -957.54MB 54.84%  github.com/Mihklz/metrixcollector/internal/repository.(*MemStorage).SaveToFile
 -268.53MB 15.38% 55.88%  -268.53MB 15.38%  bytes.growSlice
```

Отрицательные значения показывают снижение потребления памяти после оптимизаций.

### Текущие метрики производительности

```bash
$ go test -bench=BenchmarkMemStorageSaveToFile -benchmem ./internal/repository
BenchmarkMemStorageSaveToFile-8   252   4891909 ns/op   2354059 B/op   10014 allocs/op
```

### Как были достигнуты улучшения

Основная оптимизация в `MemStorage.SaveToFile`:

**До оптимизации:**
```go
var metrics []models.Metrics  // слайс без предварительного выделения
```

**После оптимизации:**
```go
metrics := make([]models.Metrics, 0, len(m.Gauges)+len(m.Counters))
```

**Эффект:**
- Предварительное выделение слайса с известной ёмкостью предотвращает множественные реаллокации при `append`
- Устранены вызовы `bytes.growSlice`, которые происходили при каждом переполнении capacity слайса
- Снижено потребление памяти за счёт уменьшения количества аллокаций и копирований данных
