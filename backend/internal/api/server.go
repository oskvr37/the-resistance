package api

import (
	"resistance/internal/config"
	"resistance/internal/store"

	"github.com/go-playground/validator/v10"
	"github.com/redis/go-redis/v9"
)

type server struct {
	Config     *config.Config
	Repository *store.Repository
	Validate   *validator.Validate
}

func NewServer(cfg *config.Config) *server {
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Username: cfg.RedisUser,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	return &server{
		Config:     cfg,
		Repository: store.NewRepository(redisClient),
		Validate:   validator.New(),
	}
}
