import { useState } from "react";
import { RoomStateResponse } from "@/hooks/room";
import { roomService, RoomSettingsRequest, GamePace } from "@/api/room";

const game_paces: GamePace[] = ["RELAXED", "STANDARD", "COMPETITIVE"];

export default function SettingsView({
  room_state,
}: {
  room_state: RoomStateResponse;
}) {
  const [settings, setSettings] = useState(room_state.settings);
  const [isSaving, setIsSaving] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const isDirty =
    JSON.stringify(settings) !== JSON.stringify(room_state.settings);

  const handleChange = (field: keyof typeof settings, value: number) => {
    setSettings((prev) => ({
      ...prev,
      [field]: value,
    }));
  };

  const handleSave = async () => {
    setIsSaving(true);
    setError(null);
    try {
      await roomService.changeRoomSettings(room_state.room_id, settings);
    } catch (err) {
      console.error(err);
      setError("Failed to update settings");
    } finally {
      setIsSaving(false);
    }
  };

  const max_players_constraint = {
    min: Math.max(room_state.members.length, 5),
    max: 10,
    label: "Max Players",
  };

  return (
    <section className="flex flex-col gap-6">
      <div className="flex justify-between items-center">
        <h2>Room Settings</h2>
        <button onClick={handleSave} disabled={!isDirty || isSaving}>
          {isSaving ? "Saving..." : "Save Changes"}
        </button>
      </div>

      {error && (
        <div className="mb-4 p-2 text-red-500 rounded text-sm">{error}</div>
      )}

      <div className="flex flex-col gap-4">
        <RangeControl
          field="max_players"
          value={settings.max_players}
          constraint={max_players_constraint}
          onChange={handleChange}
        />
      </div>
      <div className="space-y-2">
        <h2>Game Pace</h2>
        <div className="flex gap-2">
          {game_paces.map((game_pace) => (
            <button
              key={game_pace}
              onClick={() =>
                setSettings((prev) => ({
                  ...prev,
                  game_pace,
                }))
              }
              className={
                settings.game_pace == game_pace ? "border-zinc-400!" : ""
              }
            >
              {game_pace}
            </button>
          ))}
        </div>
      </div>
    </section>
  );
}

interface RangeControlProps {
  field: keyof RoomSettingsRequest;
  value: number;
  constraint: { min: number; max: number; label: string; unit?: string };
  onChange: (field: keyof RoomSettingsRequest, val: number) => void;
}

function RangeControl({
  field,
  value,
  constraint,
  onChange,
}: RangeControlProps) {
  return (
    <div className="text-sm font-light flex flex-col gap-2">
      <div className="flex justify-between">
        <label htmlFor={field}>{constraint.label}</label>
        <span>
          {value}
          {constraint.unit}
        </span>
      </div>
      <div className="flex items-center gap-3 text-zinc-400 font-medium">
        <span className="text-right">{constraint.min}</span>
        <input
          id={field}
          type="range"
          min={constraint.min}
          max={constraint.max}
          value={value}
          onChange={(e) => onChange(field, parseInt(e.target.value))}
          className="flex-1 h-2 appearance-none cursor-pointer"
        />
        <span className="text-xs w-4">{constraint.max}</span>
      </div>
    </div>
  );
}
