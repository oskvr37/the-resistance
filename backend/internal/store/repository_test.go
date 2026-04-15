package store_test

import (
	"context"
	"encoding/json"
	"resistance/internal/response"
	"resistance/internal/store"
	"strconv"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

func setupTestRepo(t *testing.T) (*store.Repository, *redis.Client) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		t.Fatalf("Redis does not respond: %v", err)
	}

	repo := store.NewRepository(client)

	return repo, client
}

type userData struct {
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

func TestRoom(t *testing.T) {
	r, c := setupTestRepo(t)
	ctx := context.Background()

	cleanup := func() {
		c.FlushDB(ctx)
	}

	t.Run("user can create new room", func(t *testing.T) {
		defer cleanup()

		userID := "user-1"
		res, err := r.RoomCreate(ctx, userID, "", "")

		assert.NoError(t, err)
		assert.NotNil(t, res.Data.RoomID)
		assert.Equal(t, response.OK, res.Code)

		roomExists := c.Exists(ctx, r.Keys.UserRoom(userID)).Val()
		assert.Equal(t, int64(1), roomExists)

		roomMembers := c.HKeys(ctx, r.Keys.RoomMembers(res.Data.RoomID)).Val()
		assert.Contains(t, roomMembers, userID)

		// default game data check
		roomData := c.HGetAll(ctx, r.Keys.Room(res.Data.RoomID)).Val()

		expectedData := map[string]string{
			"game_pace":   "STANDARD",
			"join_code":   roomData["join_code"], // We check existence, but value is random
			"max_players": "10",
			"owner":       userID,
			"status":      "LOBBY",
		}

		assert.Equal(t, expectedData, roomData)
		assert.Len(t, roomData["join_code"], 6) // Verify the code length if it's a 6-char string
	})

	t.Run("user cant create room, when already in a room", func(t *testing.T) {
		defer cleanup()

		userID := "user-1"
		currentRoomID := "old-room"

		user_data, err := json.Marshal(&userData{Username: "username", Avatar: "av"})
		assert.NoError(t, err)

		c.Set(ctx, r.Keys.UserRoom(userID), currentRoomID, 0)
		c.HSet(ctx, r.Keys.RoomMembers(currentRoomID), userID, user_data)

		res, err := r.RoomCreate(ctx, userID, "", "")
		assert.NoError(t, err)
		assert.Equal(t, response.AlreadyInRoom, res.Code)
	})

	t.Run("user can join existing room by id or join code", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "user-1"
		userJoiningID := "user-2"
		userJoiningByCodeID := "user-3"

		rcRes, err := r.RoomCreate(ctx, userOwnerID, "", "")
		assert.NoError(t, err)
		roomID := rcRes.Data.RoomID

		rjRes, err := r.RoomJoin(ctx, userJoiningID, roomID, "username", "avatar")
		assert.NoError(t, err)
		assert.Equal(t, response.OK, rjRes.Code)

		redisUserData, err := r.Client.
			HGet(ctx, r.Keys.RoomMembers(rcRes.Data.RoomID), userJoiningID).
			Bytes()
		assert.NoError(t, err)

		var joiningUserData userData
		err = json.Unmarshal(redisUserData, &joiningUserData)
		assert.NoError(t, err)

		assert.Equal(t, userData{
			Username: "username",
			Avatar:   "avatar",
		}, joiningUserData)

		joinCode, err := c.HGet(ctx, r.Keys.Room(roomID), "join_code").Result()
		assert.NoError(t, err)

		rjcRes, err := r.RoomJoinByCode(ctx, userJoiningByCodeID, joinCode, "", "")
		assert.NoError(t, err)
		assert.Equal(t, response.OK, rjcRes.Code)

		members := c.HKeys(ctx, r.Keys.RoomMembers(roomID)).Val()
		assert.Contains(t, members, userJoiningID)
		assert.Contains(t, members, userJoiningByCodeID)

		mapping := c.Get(ctx, r.Keys.UserRoom(userJoiningID)).Val()
		assert.Equal(t, roomID, mapping, "User mapping should point to the joined room ID")
	})

	t.Run("user cant join non-existent room", func(t *testing.T) {
		defer cleanup()
		res, err := r.RoomJoin(ctx, "user-1", "non-existent-room-id", "", "")
		assert.NoError(t, err)
		assert.Equal(t, response.RoomNotFound, res.Code)
	})

	t.Run("user can not join with wrong join code", func(t *testing.T) {
		defer cleanup()
		res, err := r.RoomJoinByCode(ctx, "user-1", "non-existent-join-code", "", "")
		assert.NoError(t, err)
		assert.Equal(t, response.RoomNotFound, res.Code)
	})

	t.Run("user can not join already started room", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "user-1"
		userJoiningID := "user-2"

		rcRes, err := r.RoomCreate(ctx, userOwnerID, "", "")
		assert.NoError(t, err)
		roomID := rcRes.Data.RoomID

		c.HSet(ctx, r.Keys.Room(roomID), "status", "PLAYING")
		res, err := r.RoomJoin(ctx, userJoiningID, roomID, "", "")
		assert.NoError(t, err)
		assert.Equal(t, response.RoomAlreadyStarted, res.Code)
	})

	t.Run("user can not join full room", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "user-1"
		userJoiningID := "user-2"

		rcRes, err := r.RoomCreate(ctx, userOwnerID, "", "")
		assert.NoError(t, err)
		roomID := rcRes.Data.RoomID

		c.HSet(ctx, r.Keys.Room(roomID), "max_players", 1)
		res, err := r.RoomJoin(ctx, userJoiningID, roomID, "", "")
		assert.NoError(t, err)
		assert.Equal(t, response.RoomFull, res.Code)
	})

	t.Run("user can not join room when already in", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "user-1"

		rcRes, err := r.RoomCreate(ctx, userOwnerID, "", "")
		assert.NoError(t, err)
		roomID := rcRes.Data.RoomID

		res, err := r.RoomJoin(ctx, userOwnerID, roomID, "", "")
		assert.NoError(t, err)
		assert.Equal(t, response.AlreadyInRoom, res.Code)
	})

	t.Run("user can leave lobby", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "user-1"
		anotherUserID := "user-2"

		rcRes, err := r.RoomCreate(ctx, userOwnerID, "", "")
		roomID := rcRes.Data.RoomID
		r.RoomJoin(ctx, anotherUserID, roomID, "", "")

		res, err := r.RoomLeave(ctx, userOwnerID, roomID)
		assert.NoError(t, err)
		assert.Equal(t, response.NoContent, res.Code)
		assert.NotContains(t, c.HKeys(ctx, r.Keys.RoomMembers(roomID)).Val(), userOwnerID, "owner should leave")

		assert.Empty(t, c.Get(ctx, r.Keys.UserRoom(userOwnerID)).Val(), "user room mapping should be removed")
		assert.Equal(t, anotherUserID, c.HGet(ctx, r.Keys.Room(roomID), "owner").Val(), "new owner should be set")

		res, err = r.RoomLeave(ctx, anotherUserID, roomID)
		assert.NoError(t, err)
		assert.Equal(t, response.NoContent, res.Code)

		assert.Empty(t, c.Get(ctx, r.Keys.Room(roomID)).Val(), "room should be deleted")
		assert.Empty(t, c.HKeys(ctx, r.Keys.RoomMembers(roomID)).Val(), "room members should be deleted")
	})

	t.Run("user can not leave started room", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "user-1"

		rcRes, err := r.RoomCreate(ctx, userOwnerID, "", "")
		assert.NoError(t, err)
		roomID := rcRes.Data.RoomID

		err = c.HSet(ctx, r.Keys.Room(roomID), "status", "PLAYING").Err()
		assert.NoError(t, err)

		res, err := r.RoomLeave(ctx, userOwnerID, roomID)
		assert.NoError(t, err)
		assert.Equal(t, response.CantLeaveStartedRoom, res.Code)
	})

	t.Run("user can not leave room that he is not in", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "user-1"

		_, err := r.RoomCreate(ctx, userOwnerID, "", "")
		assert.NoError(t, err)

		res, err := r.RoomLeave(ctx, userOwnerID, "non-existent-room")
		assert.NoError(t, err)
		assert.Equal(t, response.NotInThisRoom, res.Code)
	})

	t.Run("user can kick another user", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "owner-user"
		userKickedID := "kicked-user"

		room, _ := r.RoomCreate(ctx, userOwnerID, "", "")
		r.RoomJoin(ctx, userKickedID, room.Data.RoomID, "", "")

		res, err := r.RoomUserKick(ctx, userOwnerID, userKickedID, room.Data.RoomID)
		assert.NoError(t, err)
		assert.Equal(t, response.NoContent, res.Code)
		assert.False(t, r.Client.HExists(ctx, r.Keys.RoomMembers(room.Data.RoomID), userKickedID).Val(), "user should be removed from room")
	})

	t.Run("user cant kick if not room owner", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "owner-user"
		notOwnerID := "not-owner-user"

		room, _ := r.RoomCreate(ctx, userOwnerID, "", "")
		r.RoomJoin(ctx, notOwnerID, room.Data.RoomID, "", "")

		res, err := r.RoomUserKick(ctx, notOwnerID, userOwnerID, room.Data.RoomID)
		assert.NoError(t, err)
		assert.Equal(t, response.NotRoomOwner, res.Code)
	})

	t.Run("user cant self kick", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "owner-user"

		room, _ := r.RoomCreate(ctx, userOwnerID, "", "")

		res, err := r.RoomUserKick(ctx, userOwnerID, userOwnerID, room.Data.RoomID)
		assert.NoError(t, err)
		assert.Equal(t, response.CantKickYourself, res.Code)
	})

	t.Run("user cant kick non existent player", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "owner-user"
		nonExistentID := "non-existent-user"

		room, _ := r.RoomCreate(ctx, userOwnerID, "", "")

		res, err := r.RoomUserKick(ctx, userOwnerID, nonExistentID, room.Data.RoomID)
		assert.NoError(t, err)
		assert.Equal(t, response.KickedUserNotInRoom, res.Code)
	})

	t.Run("user cant kick during game", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "owner-user"
		userKickedID := "kicked-user"

		room, _ := r.RoomCreate(ctx, userOwnerID, "", "")
		r.Client.HSet(ctx, r.Keys.Room(room.Data.RoomID), "status", "PLAYING")

		res, err := r.RoomUserKick(ctx, userOwnerID, userKickedID, room.Data.RoomID)
		assert.NoError(t, err)
		assert.Equal(t, response.CantKickDuringGame, res.Code)
	})

	t.Run("room owner can change room settings", func(t *testing.T) {
		defer cleanup()

		userOwnerID := "owner-user"

		room, _ := r.RoomCreate(ctx, userOwnerID, "", "")

		res, err := r.RoomSettings(ctx, userOwnerID, room.Data.RoomID, &store.RoomSettingsRequest{
			MaxPlayers: 5,
			Pace:       "RELAXED",
		})
		assert.NoError(t, err)
		assert.Equal(t, response.NoContent, res.Code)
		maxPlayers, err := r.Client.HGet(ctx, r.Keys.Room(room.Data.RoomID), "max_players").Int()
		assert.Equal(t, 5, maxPlayers)
		gamePace := r.Client.HGet(ctx, r.Keys.Room(room.Data.RoomID), "game_pace").Val()
		assert.Equal(t, "RELAXED", gamePace)
	})

	t.Run("user cant change settings if not owner", func(t *testing.T) {
		defer cleanup()

		notOwnerID := "not-owner-user"

		room, _ := r.RoomCreate(ctx, "owner-user", "", "")

		res, err := r.RoomSettings(ctx, notOwnerID, room.Data.RoomID, &store.RoomSettingsRequest{})
		assert.NoError(t, err)
		assert.Equal(t, response.NotRoomOwner, res.Code)
	})

	t.Run("user cant change settings during game", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "", "")
		r.Client.HSet(ctx, r.Keys.Room(room.Data.RoomID), "status", "PLAYING")

		res, err := r.RoomSettings(ctx, ownerID, room.Data.RoomID, &store.RoomSettingsRequest{})
		assert.NoError(t, err)
		assert.Equal(t, response.CantChangeSettingsDuringGame, res.Code)
	})

	t.Run("user cant change settings max players below current players count", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "", "")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "", "")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "", "")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "", "")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "", "")
		r.RoomJoin(ctx, "user-5", room.Data.RoomID, "", "")
		// we got 6 players

		res, err := r.RoomSettings(ctx, ownerID, room.Data.RoomID, &store.RoomSettingsRequest{
			MaxPlayers: 5,
		})
		assert.NoError(t, err)
		assert.Equal(t, response.CantDecreasePlayerCount, res.Code)
	})

	t.Run("user can start game if is a leader", func(t *testing.T) {
		defer cleanup()

		ownerID := "user-owner"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")
		// we got 5 players

		res, err := r.RoomStart(ctx, ownerID, room.Data.RoomID)
		assert.NoError(t, err)
		assert.Equal(t, response.NoContent, res.Code)

		// 1. Check Room Hash
		roomData := r.Client.HGetAll(ctx, r.Keys.Room(room.Data.RoomID)).Val()
		assert.Equal(t, "PLAYING", roomData["status"])

		// 2. Check Game State Hash
		gameState := r.Client.HGetAll(ctx, r.Keys.GameState(room.Data.RoomID)).Val()
		assert.Equal(t, "NOMINATION", gameState["phase"])
		assert.Equal(t, "1", gameState["round"])
		assert.Equal(t, "0", gameState["failed_missions"])
		assert.NotEmpty(t, gameState["phase_expires_at"])

		// 3. Check Turn Order (List)
		order, _ := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Result()
		assert.Len(t, order, 5)
		assert.Subset(t, []string{ownerID, "user-1", "user-2", "user-3", "user-4"}, order)

		// Check if indices exist
		assert.Equal(t, "0", gameState["leader_idx"])
		assert.Contains(t, "4", gameState["hammer_idx"])

		// 4. Check Spies (Set)
		spies, _ := r.Client.SMembers(ctx, r.Keys.GameMembersSpies(room.Data.RoomID)).Result()
		assert.Len(t, spies, 2) // 5 players = 2 spies
		for _, s := range spies {
			assert.Contains(t, order, s)
		}

		// 5. Ensure cleanup of old keys (if any)
		nominees := r.Client.SCard(ctx, r.Keys.GameMembersNominated(room.Data.RoomID)).Val()
		assert.Equal(t, int64(0), nominees, "Nomination set should be empty on start")
	})

	t.Run("user cant start game if is not a leader", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")
		// we got 5 players

		res, err := r.RoomStart(ctx, "user-1", room.Data.RoomID)
		assert.NoError(t, err)
		assert.Equal(t, response.NotRoomOwner, res.Code)
	})

	t.Run("user cant start already started game", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")
		// we got 5 players

		_, err := r.RoomStart(ctx, ownerID, room.Data.RoomID)
		res, err := r.RoomStart(ctx, ownerID, room.Data.RoomID)
		assert.NoError(t, err)
		assert.Equal(t, response.RoomAlreadyStarted, res.Code)
	})

	t.Run("user can nominate members if he is leader", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")
		// we got 5 players

		_, err := r.RoomStart(ctx, ownerID, room.Data.RoomID)
		assert.NoError(t, err)

		leaderIndex, err := r.Client.HGet(ctx, r.Keys.GameState(room.Data.RoomID), "leader_idx").Int()
		assert.NoError(t, err, "Leader field should exist in Redis")

		membersOrder := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Val()

		leaderID := membersOrder[leaderIndex]
		nominatedIDs := []string{"user-1", "user-2"}

		res, err := r.RoomGameNominate(ctx, leaderID, room.Data.RoomID, nominatedIDs)

		assert.Equal(t, response.NoContent, res.Code)
		assert.ElementsMatch(t, nominatedIDs, r.Client.SMembers(ctx, r.Keys.GameMembersNominated(room.Data.RoomID)).Val())
		assert.Equal(t, "VOTING", r.Client.HGet(ctx, r.Keys.GameState(room.Data.RoomID), "phase").Val())
	})

	t.Run("user cant nominate invalid members count", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")
		// we got 5 players

		_, err := r.RoomStart(ctx, ownerID, room.Data.RoomID)
		assert.NoError(t, err)

		leaderIndex, err := r.Client.HGet(ctx, r.Keys.GameState(room.Data.RoomID), "leader_idx").Int()
		assert.NoError(t, err, "Leader field should exist in Redis")

		membersOrder := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Val()

		leaderID := membersOrder[leaderIndex]

		res, err := r.RoomGameNominate(ctx, leaderID, room.Data.RoomID, []string{"user-1", "user-2", "user-3"})

		assert.Equal(t, response.InvalidTeamSize, res.Code)
	})

	t.Run("user cant nominate duplicated members", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")
		// we got 5 players

		_, err := r.RoomStart(ctx, ownerID, room.Data.RoomID)
		assert.NoError(t, err)

		leaderIndex, err := r.Client.HGet(ctx, r.Keys.GameState(room.Data.RoomID), "leader_idx").Int()
		assert.NoError(t, err, "Leader field should exist in Redis")

		membersOrder := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Val()

		leaderID := membersOrder[leaderIndex]

		res, err := r.RoomGameNominate(ctx, leaderID, room.Data.RoomID, []string{"user-1", "user-1"})

		assert.Equal(t, response.DuplicateNominee, res.Code)
	})

	t.Run("user cant nominate members that arent in the room", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")
		// we got 5 players

		_, err := r.RoomStart(ctx, ownerID, room.Data.RoomID)
		assert.NoError(t, err)

		leaderIndex, err := r.Client.HGet(ctx, r.Keys.GameState(room.Data.RoomID), "leader_idx").Int()
		assert.NoError(t, err, "Leader field should exist in Redis")

		membersOrder := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Val()

		leaderID := membersOrder[leaderIndex]

		res, err := r.RoomGameNominate(ctx, leaderID, room.Data.RoomID, []string{"user-10", "user-1"})

		assert.Equal(t, response.NomineeNotInRoom, res.Code)
	})

	t.Run("server should pass leader after nomination timeout", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")
		// we got 5 players

		roomStart, err := r.RoomStart(ctx, ownerID, room.Data.RoomID)
		assert.NoError(t, err)

		r.GameRoomNominateTimeout(room.Data.RoomID, roomStart.Data)

		nextLeader := r.Client.HGet(ctx, r.Keys.GameState(room.Data.RoomID), "leader_idx").Val()
		assert.Equal(t, "1", nextLeader)
	})

	t.Run("server should end game after nomination timeout", func(t *testing.T) {
		defer cleanup()

		ownerID := "owner-user"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")
		// we got 5 players

		roomStart, err := r.RoomStart(ctx, ownerID, room.Data.RoomID)
		assert.NoError(t, err)

		// 5th failed nomination (leader_idx == hammer_idx)
		r.Client.HSet(ctx, r.Keys.GameState(room.Data.RoomID), "leader_idx", "4")

		r.GameRoomNominateTimeout(room.Data.RoomID, roomStart.Data)

		type RoomData struct {
			Status string `redis:"status"`
			Result string `redis:"result"`
		}

		var data RoomData
		err = r.Client.HGetAll(ctx, r.Keys.Room(room.Data.RoomID)).Scan(&data)
		assert.NoError(t, err)

		assert.Equal(t, "LOBBY", data.Status)
		assert.Equal(t, "HAMMER", data.Result)
	})

	t.Run("users can approve team voting", func(t *testing.T) {
		defer cleanup()

		ownerID := "user-0"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		// we got 5 players
		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")

		r.RoomStart(ctx, ownerID, room.Data.RoomID)

		type GameState struct {
			LeaderIndex int    `redis:"leader_idx"`
			Phase       string `redis:"phase"`
		}

		var gameState GameState
		err := r.Client.HGetAll(ctx, r.Keys.GameState(room.Data.RoomID)).Scan(&gameState)
		assert.NoError(t, err)

		membersOrder := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Val()

		leaderID := membersOrder[gameState.LeaderIndex]
		nominatedIDs := []string{"user-1", "user-2"}

		_, err = r.RoomGameNominate(ctx, leaderID, room.Data.RoomID, nominatedIDs)
		assert.NoError(t, err)

		res, err := r.RoomGameVote(ctx, "user-1", room.Data.RoomID, true)
		assert.Equal(t, response.NoContent, res.Code)

		res, err = r.RoomGameVote(ctx, "user-2", room.Data.RoomID, true)
		res, err = r.RoomGameVote(ctx, "user-3", room.Data.RoomID, true)
		res, err = r.RoomGameVote(ctx, "user-4", room.Data.RoomID, false)
		res, err = r.RoomGameVote(ctx, ownerID, room.Data.RoomID, true)
		// everyone approved this team, except "user-4"

		membersVoting := r.Client.HGetAll(ctx, r.Keys.GameMembersVoting(room.Data.RoomID)).Val()
		assert.Equal(t, "1", membersVoting[ownerID])
		assert.Equal(t, "1", membersVoting["user-1"])
		assert.Equal(t, "1", membersVoting["user-2"])
		assert.Equal(t, "1", membersVoting["user-3"])
		assert.Equal(t, "0", membersVoting["user-4"])

		err = r.Client.HGetAll(ctx, r.Keys.GameState(room.Data.RoomID)).Scan(&gameState)
		assert.NoError(t, err)
		assert.Equal(t, "MISSION", gameState.Phase)
		assert.Equal(t, 1, gameState.LeaderIndex)
	})

	t.Run("users can reject team voting", func(t *testing.T) {
		defer cleanup()

		ownerID := "user-0"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		// we got 5 players
		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")

		r.RoomStart(ctx, ownerID, room.Data.RoomID)

		type GameState struct {
			LeaderIndex int    `redis:"leader_idx"`
			Phase       string `redis:"phase"`
		}

		var gameState GameState
		err := r.Client.HGetAll(ctx, r.Keys.GameState(room.Data.RoomID)).Scan(&gameState)
		assert.NoError(t, err)

		membersOrder := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Val()

		leaderID := membersOrder[gameState.LeaderIndex]
		nominatedIDs := []string{"user-1", "user-2"}

		_, err = r.RoomGameNominate(ctx, leaderID, room.Data.RoomID, nominatedIDs)
		assert.NoError(t, err)

		res, err := r.RoomGameVote(ctx, "user-1", room.Data.RoomID, false)
		assert.Equal(t, response.NoContent, res.Code)

		res, err = r.RoomGameVote(ctx, "user-2", room.Data.RoomID, false)
		res, err = r.RoomGameVote(ctx, "user-3", room.Data.RoomID, false)
		res, err = r.RoomGameVote(ctx, "user-4", room.Data.RoomID, false)
		res, err = r.RoomGameVote(ctx, ownerID, room.Data.RoomID, false)
		// everyone rejected this team

		membersVoting, err := r.Client.HGetAll(ctx, r.Keys.GameMembersVoting(room.Data.RoomID)).Result()
		assert.NoError(t, err)
		assert.NotEmpty(t, membersVoting)

		// 2. Since it was a timeout, everyone should be "0" (false)
		for _, vote := range membersVoting {
			assert.Equal(t, "0", vote)
		}
		membersNominated := r.Client.SMembers(ctx, r.Keys.GameMembersNominated(room.Data.RoomID)).Val()
		assert.Empty(t, membersNominated)

		err = r.Client.HGetAll(ctx, r.Keys.GameState(room.Data.RoomID)).Scan(&gameState)
		assert.NoError(t, err)
		assert.Equal(t, "NOMINATION", gameState.Phase)
		assert.Equal(t, 1, gameState.LeaderIndex)
	})

	t.Run("users can reject and end up in hammer", func(t *testing.T) {
		defer cleanup()

		ownerID := "user-0"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		// we got 5 players
		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")

		r.RoomStart(ctx, ownerID, room.Data.RoomID)

		type GameState struct {
			LeaderIndex int    `redis:"leader_idx"`
			Phase       string `redis:"phase"`
		}

		var gameState GameState

		membersOrder := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Val()
		nominatedIDs := []string{"user-1", "user-2"}

		for range 5 {
			err := r.Client.HGetAll(ctx, r.Keys.GameState(room.Data.RoomID)).Scan(&gameState)
			assert.NoError(t, err)
			leaderID := membersOrder[gameState.LeaderIndex]
			r.RoomGameNominate(ctx, leaderID, room.Data.RoomID, nominatedIDs)
			r.RoomGameVote(ctx, "user-1", room.Data.RoomID, false)
			r.RoomGameVote(ctx, "user-2", room.Data.RoomID, false)
			r.RoomGameVote(ctx, "user-3", room.Data.RoomID, false)
			r.RoomGameVote(ctx, "user-4", room.Data.RoomID, false)
			r.RoomGameVote(ctx, ownerID, room.Data.RoomID, false)
			// everyone rejected this team
		}

		membersVoting := r.Client.HGetAll(ctx, r.Keys.GameMembersVoting(room.Data.RoomID)).Val()
		assert.Empty(t, membersVoting)

		membersNominated := r.Client.SMembers(ctx, r.Keys.GameMembersNominated(room.Data.RoomID)).Val()
		assert.Empty(t, membersNominated)

		// Check the Room Metadata
		roomData := r.Client.HGetAll(ctx, r.Keys.Room(room.Data.RoomID)).Val()
		assert.Equal(t, "LOBBY", roomData["status"], "Room status should be reset to LOBBY")
		assert.Equal(t, "HAMMER", roomData["result"], "Result should be recorded as HAMMER")
	})

	t.Run("server should timeout voting phase back to nomination", func(t *testing.T) {
		defer cleanup()

		ownerID := "user-0"

		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		// we got 5 players
		r.RoomJoin(ctx, "user-1", room.Data.RoomID, "1", "1")
		r.RoomJoin(ctx, "user-2", room.Data.RoomID, "2", "2")
		r.RoomJoin(ctx, "user-3", room.Data.RoomID, "3", "3")
		r.RoomJoin(ctx, "user-4", room.Data.RoomID, "4", "4")

		// room started
		r.RoomStart(ctx, ownerID, room.Data.RoomID)

		// leader nominated a team
		type GameState struct {
			LeaderIndex int    `redis:"leader_idx"`
			Phase       string `redis:"phase"`
		}

		var gameState GameState
		r.Client.HGetAll(ctx, r.Keys.GameState(room.Data.RoomID)).Scan(&gameState)
		membersOrder := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Val()
		leaderID := membersOrder[gameState.LeaderIndex]
		nominatedIDs := []string{"user-1", "user-2"}
		res, err := r.RoomGameNominate(ctx, leaderID, room.Data.RoomID, nominatedIDs)
		assert.NoError(t, err)

		// timeout run
		r.GameRoomVotingTimeout(room.Data.RoomID, res.Data)

		// every member vote should be "false"
		membersVoting, err := r.Client.HGetAll(ctx, r.Keys.GameMembersVoting(room.Data.RoomID)).Result()
		assert.NoError(t, err)
		assert.NotEmpty(t, membersVoting)

		// 2. Since it was a timeout, everyone should be "0" (false)
		for userID, vote := range membersVoting {
			assert.Equal(t, "0", vote, "User %s should have been forced to false by timeout", userID)
		}
		membersNominated := r.Client.SMembers(ctx, r.Keys.GameMembersNominated(room.Data.RoomID)).Val()
		assert.Empty(t, membersNominated)

		err = r.Client.HGetAll(ctx, r.Keys.GameState(room.Data.RoomID)).Scan(&gameState)
		assert.NoError(t, err)
		assert.Equal(t, "NOMINATION", gameState.Phase)
		assert.Equal(t, 1, gameState.LeaderIndex)
	})

	setupMissionRoom := func(playerCount int) (string, []string, []string, []string) {
		ownerID := "user-0"
		room, _ := r.RoomCreate(ctx, ownerID, "owner", "0")

		var members []string
		members = append(members, ownerID)
		for i := 1; i < playerCount; i++ {
			id := strconv.Itoa(i)
			r.RoomJoin(ctx, id, room.Data.RoomID, id, id)
			members = append(members, id)
		}

		r.RoomStart(ctx, ownerID, room.Data.RoomID)

		// Identify Spies (from the set we created in RoomStart)
		spies := r.Client.SMembers(ctx, r.Keys.GameMembersSpies(room.Data.RoomID)).Val()

		// Force a nomination and approval to get to MISSION phase
		membersOrder := r.Client.LRange(ctx, r.Keys.GameMembersOrder(room.Data.RoomID), 0, -1).Val()
		leaderID := membersOrder[0]

		// Nominate first two players
		nominees := []string{membersOrder[0], membersOrder[1]}
		r.RoomGameNominate(ctx, leaderID, room.Data.RoomID, nominees)

		// Everyone votes YES to skip to mission
		for _, m := range members {
			r.RoomGameVote(ctx, m, room.Data.RoomID, true)
		}

		return room.Data.RoomID, members, spies, nominees
	}

	t.Run("agents must pass missions", func(t *testing.T) {
		defer cleanup()
	})

	t.Run("successful mission archives result and moves to next nomination", func(t *testing.T) {
		defer cleanup()
	})

	t.Run("spy failing a mission increments failure counter", func(t *testing.T) {
		defer cleanup()
	})

	t.Run("mission timeout forces a PASS", func(t *testing.T) {
		defer cleanup()
		roomID, _, _, nominees := setupMissionRoom(5)

		// Only 1 person votes
		r.RoomGameMission(ctx, nominees[0], roomID, true)

		// Trigger timeout
		eventID := r.Client.HGet(ctx, r.Keys.GameState(roomID), "event_id").Val()
		r.RoomGameMissionTimeout(roomID, eventID)

		// Check that it succeeded (because timeout forced a Pass for the 2nd person)
		history := r.Client.LRange(ctx, r.Keys.GameMissions(roomID), 0, -1).Val()
		t.Log(history)
	})

	t.Run("third failure ends game for spies", func(t *testing.T) {
		defer cleanup()
	})
}
