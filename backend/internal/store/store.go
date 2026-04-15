// internal / store / store.go

package store

import (
	"context"
	"embed"
	"fmt"

	"github.com/redis/go-redis/v9"
)

type Repository struct {
	Client *redis.Client
	Hub    *RoomHub
	Keys   *StoreKeys
}

//go:embed functions/*.lua
var luaFiles embed.FS

func (r *Repository) registerAllFunctions(ctx context.Context) error {
	entries, err := luaFiles.ReadDir("functions")
	if err != nil {
		return err
	}

	for _, entry := range entries {
		content, err := luaFiles.ReadFile("functions/" + entry.Name())
		if err != nil {
			return err
		}

		err = r.Client.FunctionLoadReplace(ctx, string(content)).Err()
		if err != nil {
			return fmt.Errorf("failed to load %s: %w", entry.Name(), err)
		}
		fmt.Printf("Successfully registered library from %s\n", entry.Name())
	}
	return nil
}

func NewRepository(client *redis.Client) *Repository {
	keys := StoreKeys{}

	repo := &Repository{
		Client: client,
		Keys:   &keys,
		Hub:    NewRoomHub(client, &keys),
	}

	err := repo.registerAllFunctions(context.Background())
	if err != nil {
		panic(fmt.Sprintf("failed to register redis functions %v", err))
	}

	return repo
}
