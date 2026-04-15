import { useEffect, useState } from "react";
import { BASE_URL } from "@/api";
import { useAuthStore } from "@/store/auth";
import { RoomSettingsRequest } from "@/api/room";

// TODO parse roomstateresponse and add fields like:
// leader obj, hammer obj
// or helpers instead of writing oneliners everywhere

export interface RoomStateResponse {
  room_id: string;
  owner_id: string;
  join_code: string;
  status: "LOBBY" | "PLAYING";
  settings: RoomSettingsRequest;
  members: MemberInfo[];
  game: GameInfo | null;
}

export interface MemberInfo {
  id: string;
  online: boolean;
  data: MemberData;
}

export interface MemberData {
  username: string;
  avatar: string;
}

export interface GameInfo {
  phase: "NOMINATION" | "VOTING" | "MISSION" | "RESULT";
  expires: number; // Unix timestamp
  round: number;

  // Rotation
  leader_idx: number;
  hammer_idx: number;
  turn_order: string[];

  // Roles (Backend ensures this is empty for non-spies, spies can see full spy list)
  spies: string[];

  // Team selection
  nominated_team: string[];

  // Public team selection votes: Record<UserID, isApprove>
  team_voting: Record<string, boolean>;

  // Results
  missions: MissionInfo[];
}

export interface MissionInfo {
  round: number;
  is_success: boolean;
  fail_count: number;
  members: string[];
  voters: string[];
}

export const useRoomEvents = (room_id: string | undefined) => {
  const [room_state, setRoomState] = useState<RoomStateResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const setRoom = useAuthStore((state) => state.setRoom);

  useEffect(() => {
    if (!room_id) return;

    const eventSource = new EventSource(`${BASE_URL}/rooms/${room_id}/events`, {
      withCredentials: true,
    });

    eventSource.addEventListener("err", (event: MessageEvent<string>) => {
      eventSource.close();
      console.log(event.data);
      setError(event.data);
      setRoom(null);
    });

    eventSource.onmessage = (event: MessageEvent<string>) => {
      const parsedData: RoomStateResponse = JSON.parse(event.data);
      setRoomState(parsedData);
    };

    eventSource.onerror = (event) => {
      eventSource.close();
      console.error(event);
      setError("connection failed");
    };

    return () => {
      eventSource.close();
    };
  }, [setRoom, room_id]);

  return { room_state, error };
};
