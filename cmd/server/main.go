package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	_ "github.com/april1858/subscriptions-service/docs"

	"github.com/april1858/subscriptions-service/internal/config"
	"github.com/april1858/subscriptions-service/internal/dto"
	"github.com/april1858/subscriptions-service/internal/handlers"
	"github.com/april1858/subscriptions-service/internal/middleware"
	"github.com/april1858/subscriptions-service/internal/repository"
	"github.com/april1858/subscriptions-service/pkg/logger"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

func main() {
	// --- 1. КОНФИГУРАЦИЯ ---
	// Техника: Fail-Fast. Если конфиг не загрузился — падаем сразу,
	// через log.Fatalf (не через Zap, потому что логгер ещё не создан).
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	// --- 2. ЛОГГЕР ---
	// Создаётся сразу после конфига — нужен всем слоям ниже.
	// Dependency Injection: логгер передаётся в конструкторы, а не берётся из глобальной переменной.

	l, err := logger.NewLogger(false) // development
	if err != nil {
		log.Fatal("failed to create logger", zap.Error(err))
	}

	defer l.Sync() // flush буфера логов при завершении

	// --- 3. ПУЛ СОЕДИНЕНИЙ С БД ---
	// pgxpool.Pool — пул соединений pgx (нативный, не database/sql).
	// Техника: Connection Pooling. Пул переиспользует соединения,
	// не открывает новое на каждый запрос — критично для производительности.
	//
	// Почему pgxpool, а не database/sql:
	// pgx native быстрее, поддерживает бинарный протокол PostgreSQL,
	// лучше работает с типами (UUID, массивы, JSON).
	// database/sql оставляем только для миграций (через stdlib).
	pool, err := pgxpool.New(context.Background(), cfg.ConnString())
	if err != nil {
		l.Error("failed to create db pool", zap.Error(err))
		return
	}
	defer pool.Close()

	// Проверка подключения — Fail-Fast: лучше упасть здесь,
	// чем получать ошибки на каждом запросе потом.
	if err := pool.Ping(context.Background()); err != nil {
		l.Error("db ping failed", zap.Error(err))
		return
	}
	l.Info("database connected")

	// --- 4. МИГРАЦИИ ---
	// Накатываются при каждом старте сервиса.
	// Техника: Auto-migration on startup — не нужно помнить
	// «а накатил ли я миграции?» — сервис сам это делает.
	// goose.Up идемпотентен: если все миграции уже применены, он ничего не делает.
	if err := repository.RunMigrations(cfg.ConnString(), "./migrations"); err != nil {
		l.Error("migrations failed", zap.Error(err))
		return // не стартуем сервер, если миграции упали
	}
	l.Info("migrations applied successfully")

	// --- 5. HTTP-СЕРВЕР ---

	repo := repository.NewSubscriptionRepository(pool, l)
	h := handlers.NewSubscriptionHandlers(repo, l)

	r := gin.Default()
	r.Use(middleware.LogRequests(l)) // логирует все HTTP-запросы

	// 4. Регистрируем роуты с middleware и хендлером
	r.POST("/subscriptions",
		middleware.ValidateJSONBody[dto.CreateSubscriptionRequest](),
		h.Create,
	)

	r.GET("/subscriptions", h.List)
	r.GET("/subscriptions/:id", h.Get)
	r.PUT("/subscriptions/:id",
		middleware.ValidateJSONBody[dto.UpdateSubscriptionRequest](),
		h.Update,
	)

	r.DELETE("/subscriptions/:id", h.Delete)

	//g := r.Group("/aggregates")
	r.POST("aggregates/total-cost", h.GetTotalCost)

	// health
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	if err := r.Run(fmt.Sprintf(":%d", cfg.Server.Port)); err != nil {
		l.Error("failed to start server", zap.Error(err))
		os.Exit(1)
	}

	srv := &http.Server{
		Addr:    ":" + "8080",
		Handler: r,
	}

	// Запускаем в горутине — чтобы main не блокировался
	// и мог слушать сигнал завершения.
	//
	// Техника: Graceful Shutdown.
	// Сервер получает SIGINT/SIGTERM, перестаёт принимать новые запросы,
	// дорабатывает текущие, и только потом завершается.
	// Это prevents dropping in-flight requests.
	go func() {
		l.Info("server starting", zap.String("addr", ":8080"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			l.Error("http server error", zap.Error(err))
		}
	}()

	// --- 6. GRACEFUL SHUTDOWN ---
	// Блокируем main до получения сигнала от ОС.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	l.Info("shutting down server")
	if err := srv.Shutdown(context.Background()); err != nil {
		l.Error("shutdown error", zap.Error(err))
	}
}
