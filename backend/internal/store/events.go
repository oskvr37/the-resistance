// internal / store / events.go

package store

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
)

type RoomStateResponse struct {
	// room data
	RoomID   string              `json:"room_id"`
	OwnerID  string              `json:"owner_id"`
	JoinCode string              `json:"join_code"`
	Status   string              `json:"status"`
	Result   string              `json:"result"`
	Settings RoomSettingsRequest `json:"settings"`

	Members []MemberInfo `json:"members"`
	Game    *GameInfo    `json:"game"`
}

type MemberInfo struct {
	ID     string     `json:"id"`
	Online bool       `json:"online"`
	Data   MemberData `json:"data"`
}

type MemberData struct {
	Username string `json:"username"`
	Avatar   string `json:"avatar"`
}

type GameInfo struct {
	Phase          string `json:"phase"`
	PhaseExpiresAt int64  `json:"expires"`
	Round          int    `json:"round"`

	// Rotation
	LeaderIndex int      `json:"leader_idx"`
	HammerIndex int      `json:"hammer_idx"`
	TurnOrder   []string `json:"turn_order"` // The sequence of Player IDs

	// Roles (Scrubbed by Go backend)
	Spies []string `json:"spies"`

	// Team selection
	NominatedTeam []string `json:"nominated_team"`

	// Public team selection votes
	TeamVoting map[string]bool `json:"team_voting"`

	// Results
	Missions []MissionInfo `json:"missions"`
}

type MissionInfo struct {
	Round int `json:"round"`
	// Did mission succeed
	IsSuccess bool `json:"is_success"`
	// How many spies 'failed' the mission
	FailCount int `json:"fail_count"`
	// Who participated in this mission
	Members []string `json:"members"`
	// Who approved this team
	Voters []string `json:"voters"`
}

func (h *RoomHub) GetRoomState(ctx context.Context, roomID string) (*RoomStateResponse, error) {
	pipe := h.client.Pipeline()

	// 1. Queue all commands based on your key structure
	roomDataCmd := pipe.HGetAll(ctx, h.keys.Room(roomID))
	membersDataCmd := pipe.HGetAll(ctx, h.keys.RoomMembers(roomID))
	presenceCmd := pipe.ZRangeWithScores(ctx, h.keys.RoomPresence(roomID), 0, -1)

	gameStateCmd := pipe.HGetAll(ctx, h.keys.GameState(roomID))
	turnOrderCmd := pipe.LRange(ctx, h.keys.GameMembersOrder(roomID), 0, -1)
	spiesCmd := pipe.SMembers(ctx, h.keys.GameMembersSpies(roomID))
	nominatedCmd := pipe.SMembers(ctx, h.keys.GameMembersNominated(roomID))
	votingCmd := pipe.HGetAll(ctx, h.keys.GameMembersVoting(roomID))
	missionsCmd := pipe.LRange(ctx, h.keys.GameMissions(roomID), 0, -1)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return nil, err
	}

	// 2. Map Room Data
	roomMap := roomDataCmd.Val()
	if len(roomMap) == 0 {
		return nil, fmt.Errorf("room not found")
	}

	state := &RoomStateResponse{
		RoomID:   roomID,
		OwnerID:  roomMap["owner"],
		JoinCode: roomMap["join_code"],
		Status:   roomMap["status"],
		Result:   roomMap["result"],
		Settings: RoomSettingsRequest{
			MaxPlayers: castInt(roomMap["max_players"]),
			Pace:       roomMap["game_pace"],
		},
	}

	// user is online if present in ZSET
	onlineMap := make(map[string]bool)
	for _, z := range presenceCmd.Val() {
		onlineMap[z.Member.(string)] = true
	}

	for id, jsonStr := range membersDataCmd.Val() {
		var data MemberData
		json.Unmarshal([]byte(jsonStr), &data)
		state.Members = append(state.Members, MemberInfo{
			ID:     id,
			Online: onlineMap[id],
			Data:   data,
		})
	}

	// 4. Map Game Info
	gMap := gameStateCmd.Val()
	if len(gMap) > 0 {
		turnOrder := turnOrderCmd.Val()

		state.Game = &GameInfo{
			Phase:          gMap["phase"],
			PhaseExpiresAt: castInt64(gMap["phase_expires_at"]),
			Round:          castInt(gMap["round"]),
			TurnOrder:      turnOrder,
			Spies:          spiesCmd.Val(),
			NominatedTeam:  nominatedCmd.Val(),
			TeamVoting:     parseBoolMap(votingCmd.Val()),
			Missions:       make([]MissionInfo, 0),
		}

		state.Game.LeaderIndex = castInt(gMap["leader_idx"])
		state.Game.HammerIndex = castInt(gMap["hammer_idx"])

		// Map Mission History
		for _, mJSON := range missionsCmd.Val() {
			var m MissionInfo
			if err := json.Unmarshal([]byte(mJSON), &m); err == nil {
				state.Game.Missions = append(state.Game.Missions, m)
			}
		}
	}

	return state, nil
}

func castInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func castInt64(s string) int64 {
	i, _ := strconv.ParseInt(s, 10, 64)
	return i
}

func parseBoolMap(m map[string]string) map[string]bool {
	res := make(map[string]bool)
	for k, v := range m {
		res[k] = v == "true" || v == "1"
	}
	return res
}
