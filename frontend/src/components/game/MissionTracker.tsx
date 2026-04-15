import { RoomStateResponse } from "@/hooks/room";

export default function MissionTracker({
  state,
}: {
  state: RoomStateResponse;
}) {
  return (
    <div className="flex gap-2 justify-center mt-2">
      {state.game?.missions.map((mission) => {
        // Determine status color
        const bgColor = mission.is_success
          ? "bg-blue-600 border-blue-400 shadow-[0_0_10px_rgba(37,99,235,0.5)]"
          : "bg-red-600 border-red-400 shadow-[0_0_10px_rgba(220,38,38,0.5)]";

        return (
          <div
            key={mission.round}
            className={`size-12 rounded-full border-2 flex flex-col items-center justify-center transition-all ${bgColor}`}
          >
            <span className="font-bold leading-none">{mission.round}</span>
          </div>
        );
      })}
    </div>
  );
}
