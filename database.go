package ant

import (
	"context"

	"github.com/go-redis/redis"
	"github.com/jinzhu/gorm"
)

const (
	DatabaseContextKey = "database_context_key"
	keyRedis           = "redis_context_key"
)

func Database(ctx context.Context) *gorm.DB {
	return ctx.Value(DatabaseContextKey).(*gorm.DB)
}

func SetDB(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, DatabaseContextKey, db)
}

func SetupRedis(ctx context.Context, client *redis.Client) context.Context {
	return context.WithValue(ctx, keyRedis, client)
}

func Redis(ctx context.Context) *redis.Client {
	return ctx.Value(keyRedis).(*redis.Client)
}
