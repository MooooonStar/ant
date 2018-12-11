package main

import (
	"context"

	"github.com/jinzhu/gorm"
)

const (
	DatabaseContextKey = "database_context_key"
)

func Database(ctx context.Context) *gorm.DB {
	return ctx.Value(DatabaseContextKey).(*gorm.DB)
}

func SetDB(ctx context.Context, db *gorm.DB) context.Context {
	return context.WithValue(ctx, DatabaseContextKey, db)
}
