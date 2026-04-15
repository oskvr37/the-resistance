// internal / store / keys.go

package store

import "fmt"

type StoreKeys struct{}

// room

// STRING
// user:{id}:room
// map user id -> room id that user is in
func (k StoreKeys) UserRoom(userID string) string {
	return fmt.Sprintf("user:%s:room", userID)
}

// HASH
// room:{id}:member
// user id -> JSON {username: string, avatar: string}
func (k StoreKeys) RoomMembers(roomID string) string {
	return fmt.Sprintf("room:%s:members:data", roomID)
}

// ZSET
// room:{id}:presence
// member id -> timestamp
// users presence in room
func (k StoreKeys) RoomPresence(roomID string) string {
	return fmt.Sprintf("room:%s:presence", roomID)
}

// HASH
// room:{id}:data
// - owner: id
// - join_code: string
// - status: "LOBBY, PLAYING"
// - result: "SPY, AGENT, HAMMER, SURRENDER"
// - max_players: int
// - game_pace: "STANDARD, RELAXED, COMPETETIVE"

func (k StoreKeys) Room(roomID string) string {
	return fmt.Sprintf("room:%s:data", roomID)
}

// game

// HASH
// room:{id}:state
// - phase: "NOMINATION, VOTING, MISSION"
// - phase_expires_at: unix
// - event_id: string
// - member_count: int
// - round: int
// - leader_idx: index
// - hammer_idx: index
// - failed_missions: int
func (k StoreKeys) GameState(roomID string) string {
	return fmt.Sprintf("room:%s:state", roomID)
}

// LIST
// room:{id}:members:order
// order of members turn
func (k StoreKeys) GameMembersOrder(roomID string) string {
	return fmt.Sprintf("room:%s:members:order", roomID)
}

// SET
// room:{id}:members:spies
// ids of members who are spies
func (k StoreKeys) GameMembersSpies(roomID string) string {
	return fmt.Sprintf("room:%s:members:spies", roomID)
}

// SET
// room:{id}:nomination
// member ids currently proposed for the mission
func (k StoreKeys) GameMembersNominated(roomID string) string {
	return fmt.Sprintf("room:%s:members:nominated", roomID)
}

// HASH
// room:{id}:members:voting
// public votes for the team nominations
// member_id -> vote (bool)
func (k StoreKeys) GameMembersVoting(roomID string) string {
	return fmt.Sprintf("room:%s:members:voting", roomID)
}

// HASH
// room:{id}:mission_votes
// secret votes for the mission
// member_id -> vote (bool)
func (k StoreKeys) GameMissionVotes(roomID string) string {
	return fmt.Sprintf("room:%s:missions:votes", roomID)
}

// LIST
// room:{id}:missions
// results of past missions
// []JSON {round: number, is_success: bool, fails: number, members: []id, approved: []id}
func (k StoreKeys) GameMissions(roomID string) string {
	return fmt.Sprintf("room:%s:missions:results", roomID)
}

// utilities

// PUB SUB
// room:{id}:events
// room events notifications
func (k StoreKeys) RoomEvents(roomID string) string {
	return fmt.Sprintf("room:%s:events", roomID)
}

// STRING
// join_code:{join_code}
// map join code -> room id
func (k StoreKeys) JoinCode(joinCode string) string {
	return fmt.Sprintf("join_code:%s", joinCode)
}
