import { api, ApiResponse } from "@/api";

export type GamePace = "RELAXED" | "STANDARD" | "COMPETITIVE";

interface RoomCreateData {
  room_id: string;
}

export interface RoomSettingsRequest {
  max_players: number;
  game_pace: GamePace;
}

export interface GameNominationRequest {
  nominated: string[];
}

export interface GameVoteRequest {
  vote: boolean;
}

export const roomService = {
  createRoom: async () => {
    const { data } = await api.post<ApiResponse<RoomCreateData>>("/rooms");
    return data.data;
  },

  joinRoomByCode: async (join_code: string) => {
    const { data } = await api.post<ApiResponse<RoomCreateData>>(
      `/rooms/join_code/${join_code}`,
    );
    return data.data;
  },

  leaveRoom: async (room_id: string) => {
    const { data } = await api.post<ApiResponse>(`/rooms/${room_id}/leave`);
    return data.data;
  },

  kickRoomMember: async (room_id: string, user_id: string) => {
    const { data } = await api.delete<ApiResponse>(
      `/rooms/${room_id}/member/${user_id}`,
    );
    return data.data;
  },

  changeRoomSettings: async (
    room_id: string,
    settings: RoomSettingsRequest,
  ) => {
    const { data } = await api.patch<ApiResponse>(
      `/rooms/${room_id}/settings`,
      settings,
    );
    return data.data;
  },

  startRoom: async (room_id: string) => {
    const { data } = await api.post<ApiResponse>(`/rooms/${room_id}/start`);
    return data.data;
  },

  gameNominate: async (room_id: string, nominated: GameNominationRequest) => {
    const { data } = await api.post<ApiResponse>(
      `/rooms/${room_id}/game/nominate`,
      nominated,
    );
    return data.data;
  },

  gameVote: async (room_id: string, vote: GameVoteRequest) => {
    const { data } = await api.post<ApiResponse>(
      `/rooms/${room_id}/game/vote`,
      vote,
    );
    return data.data;
  },

  gameMission: async (room_id: string, vote: GameVoteRequest) => {
    const { data } = await api.post<ApiResponse>(
      `/rooms/${room_id}/game/mission`,
      vote,
    );
    return data.data;
  },
};
