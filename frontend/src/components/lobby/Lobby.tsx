import { MemberInfo, RoomStateResponse } from "@/hooks/room";
import { useAuthStore } from "@/store/auth";
import { roomService } from "@/api/room";
import Avatar from "@/components/Avatar";
import { useNavigate, useParams } from "react-router";
import SettingsView from "./Settings";

interface LobbyProps {
  room_state: RoomStateResponse;
  user_id: string;
}

export default function Lobby({ room_state, user_id }: LobbyProps) {
  const { room_id } = useParams();
  const setRoom = useAuthStore((state) => state.setRoom);

  const navigate = useNavigate();

  async function kickRoomMemberHandler(member_id: string) {
    if (!room_id) return;
    await roomService.kickRoomMember(room_id, member_id);
  }

  async function leaveRoomHandler() {
    if (!room_id) return;

    await roomService
      .leaveRoom(room_id)
      .then(() => {
        setRoom(null);
        navigate("/");
      })
      .catch((err) => {
        alert(
          err.response?.data?.code || err.message || "Failed to leave room",
        );
      });
  }

  async function startRoomHandler() {
    if (!room_id) return;

    await roomService
      .startRoom(room_id)
      .then(() => {
        console.log("room started");
      })
      .catch((err) => {
        alert(
          err.response?.data?.code || err.message || "Failed to start room",
        );
      });
  }

  return (
    <div className="space-y-4 divide-y divide-zinc-600 *:py-4 w-full">
      <section className="flex justify-between items-center">
        <p>
          Join Code:{" "}
          <span className="font-mono text-zinc-300">
            {room_state.join_code}
          </span>
        </p>
        <button onClick={leaveRoomHandler}>Leave Room</button>
      </section>

      <section className="space-y-4">
        <div className="flex justify-between items-center">
          <h2>Members</h2>
          <p>
            {room_state.members.length} / {room_state.settings.max_players}
          </p>
        </div>
        {room_state && (
          <ul className="space-y-2">
            {room_state.members.map((member) => {
              return (
                <li key={member.id}>
                  <MemberView
                    member={member}
                    canKick={room_state.owner_id == user_id}
                    isOwner={member.id == room_state.owner_id}
                    isMe={member.id == user_id}
                    kickHandler={kickRoomMemberHandler}
                  />
                </li>
              );
            })}
          </ul>
        )}
      </section>

      <section className="flex gap-2">
        <button
          disabled={room_state.members.length < 5}
          title={
            room_state.members.length < 5
              ? "Need 5 players to start!"
              : "Start game!"
          }
          onClick={startRoomHandler}
        >
          Start Game
        </button>
      </section>

      {user_id == room_state.owner_id && (
        <SettingsView room_state={room_state} />
      )}

      <section>
        <h2>Room Events</h2>
        <pre className="text-xs text-zinc-400 font-light">
          <code>{JSON.stringify(room_state, null, 2)}</code>
        </pre>
      </section>
    </div>
  );
}

interface MemberViewProps {
  member: MemberInfo;
  canKick: boolean;
  isOwner: boolean;
  isMe: boolean;
  kickHandler: (id: string) => void;
}

function MemberView({
  member,
  canKick,
  isOwner,
  isMe,
  kickHandler,
}: MemberViewProps) {
  return (
    <div className="flex items-center gap-2">
      <div className={member.online ? "opacity-100" : "opacity-50"}>
        <Avatar seed={member.data.avatar} options={{ flip: true, size: 64 }} />
      </div>
      <span
        className={`font-medium ${member.online ? "opacity-100" : "opacity-50"} transition-opacity uppercase font-mono text-zinc-300`}
      >
        {member.data.username}{" "}
        {isOwner && (
          <span title="Room Owner" className="text-sm">
            ⭐
          </span>
        )}
      </span>
      {!isMe && canKick && (
        <button
          className="font-mono text-sm uppercase"
          onClick={() => kickHandler(member.id)}
        >
          Kick
        </button>
      )}
    </div>
  );
}
