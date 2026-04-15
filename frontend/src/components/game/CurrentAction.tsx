import { RoomStateResponse } from "@/hooks/room";
import { getRequiredTeamSize } from "./utils";

interface CurrentActionProps {
  state: RoomStateResponse;
  user_id: string;
}

function GetActionText({ state, user_id }: CurrentActionProps): string {
  const { game, members } = state;
  if (!game) return "Waiting for game to start...";

  const leaderId = game.turn_order[game.leader_idx];
  const leader = members.find((m) => m.id === leaderId);
  const isUserLeader = leaderId === user_id;

  const getNominatedNames = () => {
    return members
      .filter((m) => game.nominated_team.includes(m.id))
      .map((m) => (m.id === user_id ? "You" : m.data.username))
      .join(", ");
  };

  switch (game.phase) {
    case "NOMINATION": {
      const required = getRequiredTeamSize(
        state.members.length,
        state.game?.round,
      );

      if (isUserLeader) {
        return `You are the leader. Nominate ${required} players!`;
      }

      return `${leader?.data.username || "Leader"} is nominating a team...`;
    }

    case "VOTING": {
      const names = getNominatedNames();
      return `${isUserLeader ? "You" : leader?.data.username} nominated: ${names}. Cast your vote!`;
    }

    case "MISSION": {
      const isOnMission = game.nominated_team.includes(user_id);
      if (isOnMission) {
        return "You are on the mission! Choose to Succeed or Fail.";
      }
      return `${getNominatedNames()} are on the mission...`;
    }

    case "RESULT":
      return "Mission complete. Analyzing results...";

    default:
      return "";
  }
}

export default function CurrentAction(props: CurrentActionProps) {
  const action_text = GetActionText(props);

  return (
    <section>
      <p className="uppercase">{action_text}</p>
    </section>
  );
}
