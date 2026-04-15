package response

import "net/http"

type ResponseCode string

// Error codes
const (
	InternalError                ResponseCode = "INTERNAL_ERROR"
	Unauthorized                 ResponseCode = "UNAUTHORIZED"
	RoomFull                     ResponseCode = "ROOM_FULL"
	NotEnoughPlayers             ResponseCode = "NOT_ENOUGH_PLAYERS"
	AlreadyInRoom                ResponseCode = "ALREADY_IN_ROOM"
	RoomNotFound                 ResponseCode = "ROOM_NOT_FOUND"
	CantLeaveStartedRoom         ResponseCode = "CANT_LEAVE_STARTED_ROOM"
	NotInThisRoom                ResponseCode = "NOT_IN_THIS_ROOM"
	RoomAlreadyStarted           ResponseCode = "ROOM_ALREADY_STARTED"
	NotRoomOwner                 ResponseCode = "NOT_ROOM_OWNER"
	CantKickYourself             ResponseCode = "CANT_KICK_YOURSELF"
	CantKickDuringGame           ResponseCode = "CANT_KICK_DURING_GAME"
	KickedUserNotInRoom          ResponseCode = "KICKED_USER_NOT_IN_ROOM"
	CantChangeSettingsDuringGame ResponseCode = "CANT_CHANGE_SETTINGS_DURING_GAME"
	CantDecreasePlayerCount      ResponseCode = "CANT_DECREASE_PLAYER_COUNT"
	YouGotKicked                 ResponseCode = "YOU_GOT_KICKED"
	AnotherConnection            ResponseCode = "ANOTHER_CONNECTION"
	InvalidTeamSize              ResponseCode = "INVALID_TEAM_SIZE"
	DuplicateNominee             ResponseCode = "DUPLICATE_NOMINEE"
	NomineeNotInRoom             ResponseCode = "NOMINEE_NOT_IN_ROOM"
	OK                           ResponseCode = "OK"
	NoContent                    ResponseCode = "NO_CONTENT"
)

var StatusMap = map[ResponseCode]int{
	InternalError:                http.StatusInternalServerError,
	Unauthorized:                 http.StatusUnauthorized,
	RoomFull:                     http.StatusConflict,
	NotEnoughPlayers:             http.StatusConflict,
	AlreadyInRoom:                http.StatusConflict,
	RoomNotFound:                 http.StatusNotFound,
	CantLeaveStartedRoom:         http.StatusConflict,
	NotInThisRoom:                http.StatusConflict,
	RoomAlreadyStarted:           http.StatusConflict,
	NotRoomOwner:                 http.StatusConflict,
	CantKickYourself:             http.StatusConflict,
	CantKickDuringGame:           http.StatusConflict,
	KickedUserNotInRoom:          http.StatusConflict,
	CantChangeSettingsDuringGame: http.StatusConflict,
	CantDecreasePlayerCount:      http.StatusConflict,
	InvalidTeamSize:              http.StatusConflict,
	DuplicateNominee:             http.StatusConflict,
	NomineeNotInRoom:             http.StatusConflict,
	OK:                           http.StatusOK,
	NoContent:                    http.StatusNoContent,
}
