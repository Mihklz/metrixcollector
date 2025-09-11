package main

import (
	"context"
	"log"

	"go.uber.org/fx"

	"github.com/Mihklz/metrixcollector/internal/app"
	"github.com/Mihklz/metrixcollector/internal/logger"
	"github.com/Mihklz/metrixcollector/internal/server"
)

func main() {
	// Инициализируем логгер
	if err := logger.Initialize(); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	// Обязательно сбрасываем буферы логгера при завершении программы
	defer logger.Log.Sync()

	// Создаем FX приложение с Dependency Injection
	fxApp := fx.New(
		// Провайдеры зависимостей
		fx.Provide(
			app.ProvideConfig,
			app.ProvideDatabase,
			app.ProvideStorage,
			app.ProvideFileStorageService,
			app.ProvideServer,
		),

		// Запуск сервера
		fx.Invoke(func(server *server.Server) {
			// FX автоматически управляет lifecycle через hooks
		}),

		// Lifecycle hooks для graceful startup/shutdown
		fx.Invoke(func(lc fx.Lifecycle, srv *server.Server) {
			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					// Запускаем сервер в горутине
					go func() {
						if err := srv.Run(); err != nil {
							logger.Log.Error("Server run failed")
						}
					}()
					return nil
				},
				OnStop: func(ctx context.Context) error {
					// Graceful shutdown
					return srv.Shutdown(ctx)
				},
			})
		}),
	)

	// Запускаем приложение и ждем сигналов завершения
	fxApp.Run()
}
