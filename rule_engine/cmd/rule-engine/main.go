package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"

	"rule_engine/internal/app/worker"
	"rule_engine/internal/app/worker/liveness"
	"rule_engine/internal/box"
	mongomigrate "rule_engine/internal/migrations/mongo"
	"rule_engine/internal/repository/mongo"
	"rule_engine/internal/repository/redis"
	"rule_engine/internal/service"
)

func main() {
	env, err := box.New()
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = env.Logger.Sync()
	}()

	app := worker.New(env.Config, env.Logger, env.Metrics)
	app.Start()

	db := env.Mongo.Database(env.Config.Mongo.Database)
	if env.Config.Mongo.MigrateOnStart {
		migCtx, migCancel := context.WithTimeout(context.Background(), env.Config.Mongo.ConnectTimeout)
		defer migCancel()

		owner, ok, err := mongomigrate.AcquireLock(migCtx, db, env.Config.Mongo.MigrateLockTTL)
		if err != nil {
			env.Logger.Fatal("acquire migration lock", zap.Error(err))
		}
		if ok {
			if err := mongomigrate.Run(migCtx, db); err != nil {
				env.Logger.Fatal("mongo migrations", zap.Error(err))
			}
			_ = mongomigrate.ReleaseLock(context.Background(), db, owner)
		} else {
			env.Logger.Info("migration lock held, skipping migrations")
		}
	}

	rulesRepo := mongo.NewRulesRepository(db)
	alertsRepo := mongo.NewAlertsRepository(db)
	devicesRepo := mongo.NewDevicesRepository(db)
	stateStore := redis.New(env.Redis)

	engine := service.NewRuleEngine(rulesRepo, alertsRepo, stateStore, env.Logger, time.Hour)
	livenessWorker := liveness.New(env.Config.Liveness.Interval, rulesRepo, devicesRepo, stateStore, alertsRepo, env.Metrics, env.Logger)
	workerApp := worker.NewWorker(env.Config, env.Logger, env.Metrics, env.Rabbit, engine, livenessWorker)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		if err := workerApp.Start(ctx); err != nil {
			env.Logger.Error("worker stopped", zap.Error(err))
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), env.Config.HTTPServer.ShutdownTimeout)
	defer cancel()

	if err := app.Shutdown(shutdownCtx); err != nil {
		env.Logger.Error("shutdown http server", zap.Error(err))
	}

	mongoCtx, mongoCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer mongoCancel()
	if err := env.Mongo.Disconnect(mongoCtx); err != nil {
		env.Logger.Error("mongo disconnect", zap.Error(err))
	}

	if err := env.Redis.Close(); err != nil {
		env.Logger.Error("redis close", zap.Error(err))
	}

	if err := env.RabbitConn.Close(); err != nil {
		env.Logger.Error("rabbit close", zap.Error(err))
	}
}
