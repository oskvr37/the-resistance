import { useNavigate, useParams } from "react-router";

import { useRoomEvents } from "@/hooks/room";
import { useAuthStore } from "@/store/auth";

import Lobby from "./lobby/Lobby";
import Game from "./game/Game";

export default function Room() {
  const navigate = useNavigate();
  const { room_id } = useParams();
  const { room_state, error } = useRoomEvents(room_id);
  const user = useAuthStore((state) => state.user);

  if (!user) {
    navigate(`/login?room=${room_id}`);
    return;
  }

  if (!room_state) return;

  if (error) {
    return (
      <section className="flex flex-col gap-2 items-center">
        <p>{error}</p>
        <button onClick={() => navigate("/")}>go back</button>
      </section>
    );
  }

  switch (room_state.status) {
    case "LOBBY":
      return <Lobby room_state={room_state} user_id={user.id} />;

    case "PLAYING":
      return <Game state={room_state} user_id={user.id} />;

    default:
      break;
  }
}
