package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"resistance/internal/response"
	"resistance/internal/store"
	"slices"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
)

type UserDTO struct {
	ID     string  `json:"id"`
	Name   string  `json:"username"`
	Avatar string  `json:"avatar"`
	RoomID *string `json:"room_id"`
}

// send back information about user
func (s *server) meAuthHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getUser(r)
	if !ok {
		writeError(w, response.Unauthorized)
		return
	}

	userData, err := s.Repository.UserMe(r.Context(), user.ID)
	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, &store.RepositoryResponse[UserDTO]{
		Code: response.OK,
		Data: UserDTO{
			ID:     user.ID,
			Name:   user.Name,
			Avatar: user.Avatar,
			RoomID: userData.RoomID,
		},
	})
}

func (s *server) doHeartbeatHandler(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "id")
	if roomId == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.Unauthorized)
		return
	}

	res, err := s.Repository.RoomHeartbeat(r.Context(), user.ID, roomId)
	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

func (s *server) createRoomHandler(w http.ResponseWriter, r *http.Request) {
	user, ok := getUser(r)
	if !ok {
		writeError(w, response.Unauthorized)
		return
	}

	res, err := s.Repository.RoomCreate(r.Context(), user.ID, user.Name, user.Avatar)
	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

func (s *server) joinRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "id")
	if roomId == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.InternalError)
		return
	}

	res, err := s.Repository.RoomJoin(r.Context(), user.ID, roomId, user.Name, user.Avatar)

	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

func (s *server) joinRoomByCodeHandler(w http.ResponseWriter, r *http.Request) {
	joinCode := strings.ToUpper(chi.URLParam(r, "code"))
	if joinCode == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.InternalError)
		return
	}

	res, err := s.Repository.RoomJoinByCode(r.Context(), user.ID, joinCode, user.Name, user.Avatar)

	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

func (s *server) leaveRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "id")
	if roomId == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.InternalError)
		return
	}

	res, err := s.Repository.RoomLeave(r.Context(), user.ID, roomId)

	if err != nil {
		log.Printf("leave room handler: %s", err)
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

func (s *server) kickUserRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	kickedMemberID := chi.URLParam(r, "member")
	if kickedMemberID == "" {
		http.Error(w, "no kicked member id", http.StatusBadRequest)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.InternalError)
		return
	}

	if user.ID == kickedMemberID {
		writeError(w, response.CantKickYourself)
		return
	}

	res, err := s.Repository.RoomUserKick(r.Context(), user.ID, kickedMemberID, roomID)

	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

func (s *server) settingsRoomHandler(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "id")
	if roomId == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.InternalError)
		return
	}

	var settingsRequest store.RoomSettingsRequest
	if err := json.NewDecoder(r.Body).Decode(&settingsRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.Validate.Struct(settingsRequest); err != nil {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	res, err := s.Repository.RoomSettings(r.Context(), user.ID, roomId, &settingsRequest)

	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

func (s *server) doStartGameHandler(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.InternalError)
		return
	}

	res, err := s.Repository.RoomStart(r.Context(), user.ID, roomID)

	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

type NominateRequest struct {
	// todo set min to 2, it was used for debugging
	Nominated []string `json:"nominated" validate:"required,min=1,max=5,unique,dive,required,ascii"`
}

func (s *server) gameNominateHandler(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.InternalError)
		return
	}

	var nominateRequest NominateRequest
	if err := json.NewDecoder(r.Body).Decode(&nominateRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.Validate.Struct(nominateRequest); err != nil {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	res, err := s.Repository.RoomGameNominate(r.Context(), user.ID, roomID, nominateRequest.Nominated)

	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

type VoteRequest struct {
	Vote *bool `json:"vote" validate:"required"`
}

func (s *server) gameVoteHandler(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.InternalError)
		return
	}

	var voteRequest VoteRequest
	if err := json.NewDecoder(r.Body).Decode(&voteRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.Validate.Struct(voteRequest); err != nil {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	res, err := s.Repository.RoomGameVote(r.Context(), user.ID, roomID, *voteRequest.Vote)

	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

type MissionRequest struct {
	Vote *bool `json:"vote" validate:"required"`
}

func (s *server) gameMissionHandler(w http.ResponseWriter, r *http.Request) {
	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		writeError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		writeError(w, response.InternalError)
		return
	}

	var missionRequest MissionRequest
	if err := json.NewDecoder(r.Body).Decode(&missionRequest); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if err := s.Validate.Struct(missionRequest); err != nil {
		http.Error(w, "Validation failed", http.StatusBadRequest)
		return
	}

	res, err := s.Repository.RoomGameMission(r.Context(), user.ID, roomID, *missionRequest.Vote)

	if err != nil {
		writeError(w, response.InternalError)
		return
	}

	writeResponse(w, res)
}

func (s *server) roomEventsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	roomID := chi.URLParam(r, "id")
	if roomID == "" {
		sendError(w, response.RoomNotFound)
		return
	}

	user, ok := getUser(r)
	if !ok {
		sendError(w, response.InternalError)
		return
	}

	connID, err := s.Repository.Hub.RegisterConnection(user.ID)

	if err != nil {
		sendError(w, response.AnotherConnection)
		return
	}

	defer s.Repository.Hub.UnregisterConnection(user.ID, connID)

	_, err = s.Repository.RoomJoin(r.Context(), user.ID, roomID, user.Name, user.Avatar)

	if err != nil {
		sendError(w, response.InternalError)
	}

	isMember, err := s.Repository.Client.HExists(
		r.Context(),
		s.Repository.Keys.RoomMembers(roomID),
		user.ID).
		Result()

	if !isMember {
		sendError(w, response.NotInThisRoom)
		return
	}

	clientChan := s.Repository.Hub.Subscribe(roomID)
	defer s.Repository.Hub.Unsubscribe(roomID, clientChan)

	heartbeatTicker := time.NewTicker(10 * time.Second)
	defer heartbeatTicker.Stop()

	initialState, err := s.Repository.Hub.GetRoomState(r.Context(), roomID)

	if err != nil {
		log.Printf("ERR GetRoomState: %s - %s", roomID, err)
		sendError(w, response.InternalError)
		return
	}

	if initialState == nil {
		sendError(w, response.RoomNotFound)
		return
	}

	// initial actions
	s.Repository.RoomHeartbeat(r.Context(), user.ID, roomID)
	sendRoomState(w, initialState, user.ID)

	for {
		select {
		// user closed tab or refreshed
		case <-r.Context().Done():
			s.Repository.RoomLeave(context.Background(), user.ID, roomID)
			return

		// keep-alive
		case <-heartbeatTicker.C:
			fmt.Fprintf(w, ": keep-alive\n\n")
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			s.Repository.RoomHeartbeat(r.Context(), user.ID, roomID)

		// new room state
		case state, ok := <-clientChan:
			if !ok {
				return
			}

			// user got removed from room
			if !isUserInRoom(state, user.ID) {
				sendError(w, response.YouGotKicked)
				return
			}

			// user is in room, send room state
			sendRoomState(w, state, user.ID)
		}
	}
}

func sendError(w http.ResponseWriter, errType response.ResponseCode) {
	fmt.Fprintf(w, "event: err\ndata: %s\n\n", errType)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func sendRoomState(w http.ResponseWriter, state *store.RoomStateResponse, userID string) {
	// 1. Create a shallow copy of the state to avoid mutating the original
	view := *state

	// 2. If the game has started, handle role privacy
	if view.Game != nil {
		// Create a copy of the Game object so we don't mess with other goroutines
		gameView := *view.Game

		isSpy := slices.Contains(gameView.Spies, userID)

		// If user isn't a spy, hide the spy list
		if !isSpy {
			gameView.Spies = []string{}
		}

		view.Game = &gameView
	}

	// 3. Marshal and stream
	payload, err := json.Marshal(view)
	if err != nil {
		return
	}

	fmt.Fprintf(w, "data: %s\n\n", payload)

	if f, ok := w.(http.Flusher); ok {
		f.Flush()
	}
}

func isUserInRoom(room *store.RoomStateResponse, userID string) bool {
	for _, member := range room.Members {
		if member.ID == userID {
			return true
		}
	}
	return false
}
