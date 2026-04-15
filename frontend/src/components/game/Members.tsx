import { useEffect, useState } from "react";
import { RoomStateResponse, MemberInfo } from "@/hooks/room";
import CurrentAction from "./CurrentAction";
import Avatar from "../Avatar";
import MissionTracker from "./MissionTracker";
import { roomService } from "@/api/room";
import { getRequiredTeamSize } from "./utils";

interface RoomMembersProps {
  state: RoomStateResponse;
  user_id: string;
}

export default function RoomMembers({ state, user_id }: RoomMembersProps) {
  const { game, members, room_id } = state;

  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const [vote, setVote] = useState<boolean | null>(null);

  useEffect(() => {
    const isLeader = game?.turn_order[game.leader_idx] === user_id;
    const isNominationPhase = game?.phase === "NOMINATION";

    // If the game moves to VOTING or another phase,
    // or if the leadership passes to someone else, clear the local picks.
    if (!isNominationPhase || !isLeader) {
      // fixme, this can trigger cascading renders
      // eslint-disable-next-line react-hooks/set-state-in-effect
      setSelectedIds([]);
    }

    setVote(null);

    // Also clear if the round changes (just in case)
  }, [game?.phase, game?.round, game?.leader_idx, user_id, game?.turn_order]);

  useEffect(() => {
    // fixme, this can trigger cascading renders
    // eslint-disable-next-line react-hooks/set-state-in-effect
    setVote(null);
  }, [selectedIds]);

  if (!game) return null;

  const isLeader = game.turn_order[game.leader_idx] === user_id;
  const isNominationPhase = game.phase === "NOMINATION";
  const canNominate = isLeader && isNominationPhase;

  const requiredCount = getRequiredTeamSize(
    state.members.length,
    state.game?.round,
  );

  const toggleNomination = (id: string) => {
    if (!canNominate) return;

    setSelectedIds((prev) =>
      prev.includes(id)
        ? prev.filter((i) => i !== id)
        : prev.length < requiredCount
          ? [...prev, id]
          : prev,
    );
  };

  const displayMembers = game.turn_order
    .map((id) => members.find((m) => m.id === id))
    .filter(Boolean) as MemberInfo[];

  const totalMembers = displayMembers.length;
  const radius = 340;

  return (
    <div className="relative w-3xl h-192 mx-auto flex items-center justify-center">
      <div className="absolute z-10 text-center space-y-4">
        <h2 className="text-zinc-500 uppercase text-xs tracking-widest font-bold">
          Round {game.round}
        </h2>
        <p className="text-xl font-black">{game.phase}</p>
        <CurrentAction state={state} user_id={user_id} />
        <MissionTracker state={state} />
        {canNominate && (
          <button
            // todo disabled button
            onClick={async () => {
              await roomService.gameNominate(room_id, {
                nominated: selectedIds,
              });
            }}
          >
            Nominate
          </button>
        )}
        {game.phase == "VOTING" &&
          (vote === null ? (
            <div className="space-x-2">
              <button
                className="text-green-400!"
                onClick={async () => {
                  await roomService
                    .gameVote(room_id, { vote: true })
                    .then(() => setVote(true));
                }}
              >
                Accept
              </button>
              <button
                className="text-red-400!"
                onClick={async () => {
                  await roomService
                    .gameVote(room_id, { vote: false })
                    .then(() => setVote(false));
                }}
              >
                Reject
              </button>
            </div>
          ) : (
            <p>{vote ? "Accepted" : "Rejected"}</p>
          ))}
        {game.phase == "MISSION" && game.nominated_team.includes(user_id) && (
          <div className="space-x-2">
            <button
              className="text-green-400!"
              onClick={async () => {
                await roomService.gameMission(room_id, { vote: true });
              }}
            >
              Succeed
            </button>
            <button
              className="text-red-400!"
              onClick={async () => {
                await roomService.gameMission(room_id, { vote: false });
              }}
            >
              Fail
            </button>
          </div>
        )}
      </div>

      {/* The Table/Circle */}
      <div className="absolute inset-10 border-2 border-zinc-800 rounded-full border-dashed" />

      {displayMembers.map((member, index) => {
        const angle = (index / totalMembers) * 360;
        const flip_avatar = angle > 60 && angle < 270;

        return (
          <div
            key={member.id}
            className={`absolute transition-all duration-500 ${canNominate ? "cursor-pointer hover:opacity-50" : ""}`}
            style={{
              transform: `rotate(${angle}deg) translate(${radius}px) rotate(-${angle}deg)`,
            }}
            onClick={() => toggleNomination(member.id)}
          >
            <MemberCard
              state={state}
              member={member}
              user_id={user_id}
              flip_avatar={flip_avatar}
              is_selected={selectedIds.includes(member.id)}
            />
          </div>
        );
      })}
    </div>
  );
}
function MemberCard({
  state,
  member,
  user_id,
  flip_avatar,
  is_selected,
}: {
  state: RoomStateResponse;
  member: MemberInfo;
  user_id: string;
  flip_avatar: boolean;
  is_selected: boolean;
}) {
  const { game } = state;
  const isLeader = game?.turn_order[game.leader_idx] === member.id;
  const isSpy = game?.spies.includes(member.id);
  const isNominated = game?.nominated_team.includes(member.id) || is_selected;
  const hasVoted =
    game?.phase === "VOTING" && game.team_voting[member.id] !== undefined;
  const hasHammer = game?.turn_order[game.hammer_idx] === member.id;

  return (
    <div className="flex flex-col items-center group">
      <div className="h-6 flex gap-1 mb-1">
        {isLeader && (
          <span title="Leader" className="animate-bounce">
            👑
          </span>
        )}
        {hasHammer && (
          <span title="Hammer" className="animate-bounce">
            🔨
          </span>
        )}
        {hasVoted && (
          <span className="text-green-500 text-xs font-bold">READY</span>
        )}
      </div>

      <div
        className={`
        relative p-1 rounded-full transition-all duration-300 ring-2
        ${isNominated ? "ring-yellow-500" : "ring-zinc-700"}
        ${!member.online ? "grayscale opacity-50" : ""}
      `}
      >
        <Avatar
          seed={member.data.avatar}
          options={{ size: 96, flip: flip_avatar }}
        />
      </div>

      <div className="mt-2">
        <p
          className={`text-sm font-bold truncate max-w-20 ${isSpy && "text-red-500"} ${member.id === user_id && "text-blue-400"}`}
        >
          {member.data.username}
        </p>
      </div>
    </div>
  );
}
