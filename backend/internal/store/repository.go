// internal / store / repository.go

package store

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"resistance/internal/response"
)

type RepositoryResponse[T any] struct {
	Code response.ResponseCode `json:"code,omitempty"`
	Data T                     `json:"data,omitempty"`
}

type userMe struct {
	RoomID *string
}

func (r *Repository) UserMe(ctx context.Context, userID string) (*userMe, error) {
	roomID, err := r.Client.Get(ctx, r.Keys.UserRoom(userID)).Result()

	if err == redis.Nil {
		return &userMe{
			RoomID: nil,
		}, nil
	}

	roomExists := r.Client.Exists(ctx, r.Keys.Room(roomID)).Val()

	if err != nil {
		return nil, err
	}

	if roomExists == 0 {
		return &userMe{
			RoomID: nil,
		}, nil
	}

	return &userMe{RoomID: &roomID}, nil
}

func (r *Repository) RoomHeartbeat(ctx context.Context, userID, roomID string) (*RepositoryResponse[any], error) {
	keys := []string{
		r.Keys.RoomPresence(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.UserRoom(userID),
		r.Keys.Room(roomID),
		r.Keys.RoomMembers(roomID),
		r.Keys.RoomMembers(userID)}

	args := []any{
		userID,
		roomID,
	}

	res, err := r.Client.FCall(ctx, "room_heartbeat", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR room_heartbeat %s", err)
		return nil, err
	}

	status, statusOk := res[0].(string)
	code, codeOk := res[1].(string)

	if !statusOk || !codeOk {
		return &RepositoryResponse[any]{
			Code: response.InternalError,
		}, nil
	}

	if status == "ERR" {
		return &RepositoryResponse[any]{
			Code: response.ResponseCode(code),
		}, nil
	}

	return &RepositoryResponse[any]{
		Code: response.ResponseCode(code),
	}, nil
}

type RoomCreateData struct {
	RoomID string `json:"room_id"`
}

func (r *Repository) RoomCreate(ctx context.Context, userID, username, avatar string) (*RepositoryResponse[*RoomCreateData], error) {
	newRoomId := uuid.New().String()
	joinCode := randomCapsString(6)

	keys := []string{
		r.Keys.UserRoom(userID),
		r.Keys.RoomMembers(newRoomId),
		r.Keys.RoomEvents(newRoomId),
		r.Keys.Room(newRoomId),
		r.Keys.JoinCode(joinCode),
		r.Keys.RoomMembers(userID),
		r.Keys.RoomPresence(newRoomId)}
	args := []any{userID, newRoomId, joinCode, username, avatar}

	res, err := r.Client.FCall(ctx, "room_create", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR room_create %s", err)
		return nil, err
	}

	status, statusOk := res[0].(string)
	code, codeOk := res[1].(string)

	if !statusOk || !codeOk {
		return &RepositoryResponse[*RoomCreateData]{
			Code: response.InternalError,
		}, nil
	}

	if status == "ERR" {
		return &RepositoryResponse[*RoomCreateData]{
			Code: response.ResponseCode(code),
		}, nil
	}

	return &RepositoryResponse[*RoomCreateData]{
		Code: response.ResponseCode(code),
		Data: &RoomCreateData{
			RoomID: newRoomId,
		},
	}, nil
}

type RoomData struct {
	ID string `json:"room_id"`
}

// join room by `roomID`
func (r *Repository) RoomJoin(ctx context.Context, userID, roomID, username, avatar string) (*RepositoryResponse[*RoomData], error) {
	keys := []string{
		r.Keys.UserRoom(userID),
		r.Keys.RoomMembers(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.Room(roomID),
		r.Keys.RoomMembers(userID),
		r.Keys.RoomPresence(roomID)}
	args := []any{userID, roomID, username, avatar, time.Now().Unix()}

	res, err := r.Client.FCall(ctx, "room_join", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR room_join %s", err)
		return nil, err
	}

	_, statusOk := res[0].(string)
	code, codeOk := res[1].(string)

	if !statusOk || !codeOk {
		return &RepositoryResponse[*RoomData]{
			Code: response.InternalError,
		}, nil
	}

	return &RepositoryResponse[*RoomData]{Code: response.ResponseCode(code), Data: &RoomData{
		ID: roomID,
	}}, nil
}

// join room by `joinCode`
func (r *Repository) RoomJoinByCode(ctx context.Context, userID, joinCode, username, avatar string) (*RepositoryResponse[*RoomData], error) {
	roomID, err := r.Client.Get(ctx, r.Keys.JoinCode(joinCode)).Result()

	if err == redis.Nil {
		return &RepositoryResponse[*RoomData]{Code: response.RoomNotFound}, nil
	}

	if err != nil {
		return nil, err
	}

	return r.RoomJoin(ctx, userID, roomID, username, avatar)
}

func (r *Repository) RoomLeave(ctx context.Context, userID, roomID string) (*RepositoryResponse[any], error) {
	keys := []string{
		r.Keys.UserRoom(userID),
		r.Keys.RoomMembers(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.Room(roomID),
		r.Keys.RoomPresence(roomID)}
	args := []any{userID, roomID}

	res, err := r.Client.FCall(ctx, "room_leave", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR room_leave %s", err)
		return nil, err
	}

	_, statusOk := res[0].(string)
	code, codeOk := res[1].(string)

	if !statusOk || !codeOk {
		return &RepositoryResponse[any]{
			Code: response.InternalError,
		}, nil
	}

	return &RepositoryResponse[any]{Code: response.ResponseCode(code)}, nil
}

func (r *Repository) RoomUserKick(ctx context.Context, ownerID, kickedUserID, roomID string) (*RepositoryResponse[any], error) {
	keys := []string{
		r.Keys.Room(roomID),
		r.Keys.RoomMembers(roomID),
		r.Keys.RoomPresence(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.UserRoom(kickedUserID)}
	args := []any{ownerID, kickedUserID}

	res, err := r.Client.FCall(ctx, "room_user_kick", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR room_user_kick %s", err)
		return nil, err
	}

	_, statusOk := res[0].(string)
	code, codeOk := res[1].(string)

	if !statusOk || !codeOk {
		return &RepositoryResponse[any]{
			Code: response.InternalError,
		}, nil
	}

	return &RepositoryResponse[any]{
		Code: response.ResponseCode(code),
	}, nil
}

type RoomSettingsRequest struct {
	MaxPlayers int    `json:"max_players" validate:"required,min=5,max=10"`
	Pace       string `json:"game_pace" validate:"required,oneof=RELAXED STANDARD COMPETITIVE"`
}

func (r *Repository) RoomSettings(ctx context.Context, ownerID, roomID string, settings *RoomSettingsRequest) (*RepositoryResponse[any], error) {
	keys := []string{
		r.Keys.Room(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.RoomMembers(roomID)}
	args := []any{ownerID, settings.MaxPlayers, settings.Pace}

	res, err := r.Client.FCall(ctx, "room_settings", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR room_settings %s", err)
		return nil, err
	}

	_, statusOk := res[0].(string)
	code, codeOk := res[1].(string)

	if !statusOk || !codeOk {
		return &RepositoryResponse[any]{
			Code: response.InternalError,
		}, nil
	}

	return &RepositoryResponse[any]{
		Code: response.ResponseCode(code),
	}, nil
}

func (r *Repository) RoomStart(ctx context.Context, ownerID, roomID string) (*RepositoryResponse[string], error) {
	eventID := uuid.New().String()

	keys := []string{
		r.Keys.Room(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.RoomMembers(roomID),
		r.Keys.GameMembersSpies(roomID),
		r.Keys.GameMembersOrder(roomID),
		r.Keys.GameState(roomID),
		r.Keys.GameMissions(roomID),
	}
	args := []any{ownerID, eventID}

	res, err := r.Client.FCall(ctx, "room_start", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR room_start %s", err)
		return nil, err
	}

	log.Printf("room_start %s", err)

	status, statusOk := res[0].(string)
	code, codeOk := res[1].(string)

	if !statusOk || !codeOk {
		return &RepositoryResponse[string]{
			Code: response.InternalError,
			Data: eventID,
		}, nil
	}

	if status == "ERR" {
		return &RepositoryResponse[string]{
			Code: response.ResponseCode(code),
			Data: eventID,
		}, nil
	}

	expireAt, _ := res[2].(int64)
	duration := max(time.Until(time.Unix(int64(expireAt), 0)), 0)

	// game started
	// run timeout on nomination phase

	time.AfterFunc(duration, func() {
		r.GameRoomNominateTimeout(roomID, eventID)
	})

	return &RepositoryResponse[string]{
		Code: response.ResponseCode(code),
		Data: eventID,
	}, nil
}

func (r *Repository) RoomGameNominate(ctx context.Context, userID, roomID string, nominatedID []string) (*RepositoryResponse[string], error) {
	nominatedJSON, err := json.Marshal(nominatedID)
	if err != nil {
		return nil, err
	}

	eventID := uuid.New().String()

	keys := []string{
		r.Keys.GameState(roomID),
		r.Keys.RoomMembers(roomID),
		r.Keys.GameMembersNominated(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.Room(roomID),
		r.Keys.GameMembersOrder(roomID),
		r.Keys.GameMembersVoting(roomID),
	}
	args := []any{userID, string(nominatedJSON), eventID}

	res, err := r.Client.FCall(ctx, "game_nominate", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR game_nominate %s", err)
		return nil, err
	}

	log.Printf("game_nominate %s", err)

	status, statusOk := res[0].(string)
	code, codeOk := res[1].(string)

	if !statusOk || !codeOk {
		return &RepositoryResponse[string]{
			Code: response.InternalError,
		}, nil
	}

	if status == "ERR" {
		return &RepositoryResponse[string]{
			Code: response.ResponseCode(code),
		}, nil
	}

	expireAt, _ := res[2].(int64)
	duration := max(time.Until(time.Unix(int64(expireAt), 0)), 0)

	// user made successful nomination
	// run timeout on voting phase

	time.AfterFunc(duration, func() {
		r.GameRoomVotingTimeout(roomID, eventID)
	})

	return &RepositoryResponse[string]{
		Code: response.ResponseCode(code),
		Data: eventID,
	}, nil
}

func (r *Repository) GameRoomNominateTimeout(roomID, eventID string) {
	nextEventID := uuid.New().String()
	keys := []string{
		r.Keys.GameState(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.Room(roomID),
		r.Keys.GameMembersOrder(roomID)}
	args := []any{eventID, nextEventID}

	res, err := r.Client.FCall(context.Background(), "game_nominate_timeout", keys, args...).Slice()
	log.Printf("game_nominate_timeout %s", res)

	if err != nil {
		return
	}

	status, statusOk := res[0].(string)
	_, codeOk := res[1].(string)

	if !statusOk || !codeOk {
		return
	}

	if status == "ERR" {
		return
	}

	expireAt, _ := res[2].(int64)
	duration := max(time.Until(time.Unix(int64(expireAt), 0)), 0)

	// run next nomination timeout for case when next leader does not nominate

	time.AfterFunc(duration, func() {
		r.GameRoomNominateTimeout(roomID, nextEventID)
	})
}

func (r *Repository) RoomGameVote(ctx context.Context, userID, roomID string, vote bool) (*RepositoryResponse[any], error) {
	voteVal := "0"
	if vote {
		voteVal = "1"
	}

	nextEventID := uuid.New().String()

	keys := []string{
		r.Keys.GameState(roomID),
		r.Keys.GameMembersVoting(roomID),
		r.Keys.RoomMembers(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.Room(roomID),
		r.Keys.GameMembersNominated(roomID),
	}
	args := []any{userID, voteVal, nextEventID}

	res, err := r.Client.FCall(ctx, "game_vote", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR game_vote %s", err)
		return nil, err
	}

	log.Printf("game_vote %s", res)

	status, statusOk := res[0].(string)
	code, codeOk := res[1].(string)
	expiresAt, expiresOk := res[2].(int64)

	if !statusOk || !codeOk || !expiresOk {
		return &RepositoryResponse[any]{
			Code: response.InternalError,
		}, nil
	}

	if status == "ERR" {
		return &RepositoryResponse[any]{
			Code: response.ResponseCode(code),
		}, nil
	}

	duration := max(time.Until(time.Unix(int64(expiresAt), 0)), 0)

	time.AfterFunc(duration, func() {
		switch code {
		case "TEAM_APPROVED":
			// run mission timeout
			r.RoomGameMissionTimeout(roomID, nextEventID)
		case "TEAM_REJECTED":
			// run nominate timeout
			r.GameRoomNominateTimeout(roomID, nextEventID)
		}
	})

	return &RepositoryResponse[any]{
		Code: response.NoContent,
	}, nil
}

func (r *Repository) GameRoomVotingTimeout(roomID, eventID string) {
	nextEventID := uuid.New().String()
	keys := []string{
		r.Keys.GameState(roomID),
		r.Keys.GameMembersVoting(roomID),
		r.Keys.RoomMembers(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.Room(roomID),
		r.Keys.GameMembersNominated(roomID),
	}
	args := []any{eventID, nextEventID}

	res, err := r.Client.FCall(context.Background(), "game_vote_timeout", keys, args...).Slice()

	if err != nil {
		return
	}

	log.Printf("game_vote_timeout %s", res)

	_, statusOk := res[0].(string)
	code, codeOk := res[1].(string)
	expiresAt, expiresOk := res[2].(int64)

	if !statusOk || !codeOk || !expiresOk {
		return
	}

	duration := max(time.Until(time.Unix(int64(expiresAt), 0)), 0)

	time.AfterFunc(duration, func() {
		switch code {
		case "TEAM_APPROVED":
			// run mission timeout
			r.RoomGameMissionTimeout(roomID, nextEventID)
		case "TEAM_REJECTED":
			// run nominate timeout
			r.GameRoomNominateTimeout(roomID, nextEventID)
		}
	})
}

func (r *Repository) RoomGameMission(ctx context.Context, userID, roomID string, vote bool) (*RepositoryResponse[any], error) {
	voteVal := "0"
	if vote {
		voteVal = "1"
	}
	nextEventID := uuid.New().String()

	keys := []string{
		r.Keys.GameState(roomID),
		r.Keys.GameMissionVotes(roomID),
		r.Keys.RoomMembers(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.Room(roomID),
		r.Keys.GameMembersNominated(roomID),
		r.Keys.GameMissions(roomID),
		r.Keys.GameMembersSpies(roomID),
		r.Keys.GameMembersVoting(roomID),
	}
	args := []any{userID, voteVal, nextEventID}

	res, err := r.Client.FCall(ctx, "game_mission", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR game_mission %s", err)
		return nil, err
	}

	log.Printf("game_mission %s", res)

	status, statusOk := res[0].(string)
	code, codeOk := res[1].(string)
	expiresAt, expiresOk := res[2].(int64)

	if !statusOk || !codeOk || !expiresOk {
		return &RepositoryResponse[any]{
			Code: response.InternalError,
		}, nil
	}

	if status == "ERR" {
		return &RepositoryResponse[any]{
			Code: response.ResponseCode(code),
		}, nil
	}

	if code == "GAME_MISSION_DONE" {
		duration := max(time.Until(time.Unix(int64(expiresAt), 0)), 0)

		time.AfterFunc(duration, func() {
			r.GameRoomNominateTimeout(roomID, nextEventID)
		})
	}

	return &RepositoryResponse[any]{
		Code: response.NoContent,
	}, nil
}

func (r *Repository) RoomGameMissionTimeout(roomID, eventID string) {
	nextEventID := uuid.New().String()
	keys := []string{
		r.Keys.GameState(roomID),
		r.Keys.GameMissionVotes(roomID),
		r.Keys.RoomMembers(roomID),
		r.Keys.RoomEvents(roomID),
		r.Keys.Room(roomID),
		r.Keys.GameMembersNominated(roomID),
		r.Keys.GameMissions(roomID),
		r.Keys.GameMembersVoting(roomID),
	}
	args := []any{eventID, nextEventID}

	res, err := r.Client.FCall(context.Background(), "game_mission_timeout", keys, args...).Slice()
	if err != nil {
		log.Printf("ERROR game_mission %s", err)
		return
	}

	log.Printf("game_mission_timeout %s", res)

	status, statusOk := res[0].(string)
	code, codeOk := res[1].(string)
	expiresAt, expiresOk := res[2].(int64)

	if !statusOk || !codeOk || !expiresOk {
		return
	}

	if status == "ERR" {
		return
	}

	if code == "GAME_MISSION_DONE" {
		duration := max(time.Until(time.Unix(int64(expiresAt), 0)), 0)

		time.AfterFunc(duration, func() {
			r.GameRoomNominateTimeout(roomID, nextEventID)
		})
	}
}
