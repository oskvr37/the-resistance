package store

import (
	"context"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type roomContext struct {
	clients map[chan *RoomStateResponse]bool
	stop    chan struct{}
}

type RoomHub struct {
	sync.RWMutex
	rooms   map[string]*roomContext
	client  *redis.Client
	keys    StoreKeys
	conns   map[string]string
	connsMu sync.Mutex
}

func NewRoomHub(client *redis.Client, keys *StoreKeys) *RoomHub {
	return &RoomHub{
		rooms:  make(map[string]*roomContext),
		client: client,
		keys:   *keys,
		conns:  make(map[string]string),
	}
}

func (h *RoomHub) RegisterConnection(userID string) (string, error) {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()

	if _, exists := h.conns[userID]; exists {
		return "", errors.New("already_connected")
	}

	connID := uuid.New().String()

	h.conns[userID] = connID

	return connID, nil
}

func (h *RoomHub) UnregisterConnection(userID string, connID string) {
	h.connsMu.Lock()
	defer h.connsMu.Unlock()

	if currentID, exists := h.conns[userID]; exists && currentID == connID {
		delete(h.conns, userID)
	}
}

func (h *RoomHub) Subscribe(roomID string) chan *RoomStateResponse {
	h.Lock()
	defer h.Unlock()

	if _, ok := h.rooms[roomID]; !ok {
		h.rooms[roomID] = &roomContext{
			clients: make(map[chan *RoomStateResponse]bool),
			stop:    make(chan struct{}),
		}
		go h.listenRedis(roomID)
	}

	clientChan := make(chan *RoomStateResponse, 1)
	h.rooms[roomID].clients[clientChan] = true

	return clientChan
}

func (h *RoomHub) Unsubscribe(roomID string, clientChan chan *RoomStateResponse) {
	h.Lock()
	defer h.Unlock()

	room, ok := h.rooms[roomID]
	if !ok {
		return
	}

	delete(room.clients, clientChan)
	close(clientChan)

	if len(room.clients) == 0 {
		close(room.stop)
		delete(h.rooms, roomID)
	}
}

func (h *RoomHub) listenRedis(roomID string) {
	h.RLock()
	stopChan := h.rooms[roomID].stop
	h.RUnlock()

	pubsub := h.client.Subscribe(context.Background(), h.keys.RoomEvents(roomID))
	defer pubsub.Close()

	ch := pubsub.Channel()

	refreshTicker := time.NewTicker(2 * time.Minute)
	defer refreshTicker.Stop()

	h.refreshRoom(context.Background(), roomID)

	for {
		select {
		case <-ch:
			// get latest room state after room event
			state, _ := h.GetRoomState(context.Background(), roomID)

			// refresh room when event happened
			h.refreshRoom(context.Background(), roomID)

			// send state to every connected SSE client
			h.RLock()
			room, ok := h.rooms[roomID]
			if ok {
				for c := range room.clients {
					select {
					case c <- state:
					default:
					}
				}
			}
			h.RUnlock()

		// refresh room if anyone is connected
		case <-refreshTicker.C:
			h.refreshRoom(context.Background(), roomID)

		case <-stopChan:
			return
		}
	}
}

func (h *RoomHub) refreshRoom(ctx context.Context, roomID string) {
	// we need to get room's join code to send it as key to the redis function
	joinCode := h.client.HGet(ctx, h.keys.Room(roomID), "join_code").Val()

	err := h.client.FCall(ctx, "room_refresh", []string{
		h.keys.JoinCode(joinCode),
		h.keys.Room(roomID),
		h.keys.RoomMembers(roomID),
		h.keys.RoomPresence(roomID),
		h.keys.GameState(roomID),
		h.keys.GameMembersOrder(roomID),
		h.keys.GameMembersSpies(roomID),
		h.keys.GameMembersNominated(roomID),
		h.keys.GameMembersVoting(roomID),
		h.keys.GameMissionVotes(roomID),
		h.keys.GameMissions(roomID),
	}).Err()

	if err != nil {
		log.Printf("Could not refresh room %s - %s", roomID, err)
	}
}
