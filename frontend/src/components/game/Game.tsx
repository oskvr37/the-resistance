import { RoomStateResponse } from "@/hooks/room";
import { useCountdown } from "@/hooks/countdown";

import RoomMembers from "./Members";

interface GameProps {
  state: RoomStateResponse;
  user_id: string;
}

export default function Game({ state, user_id }: GameProps) {
  const time = useCountdown(state.game?.expires ?? 0);

  if (!state.game) return;

  return (
    <div>
      <p>Time {time.toFixed(1)}</p>
      <RoomMembers state={state} user_id={user_id} />
      <code>
        <pre>{JSON.stringify(state, null, 2)}</pre>
      </code>
    </div>
  );
}
