import { useAuthStore } from "@/store/auth";
import { roomService } from "@/api/room";
import { useState, useEffect } from "react";
import { useNavigate } from "react-router";

export default function JoinCreateRoom() {
  const navigate = useNavigate();
  const is_authenticated = useAuthStore((state) => state.is_authenticated);
  const user_room_id = useAuthStore((state) => state.user?.room_id);
  const setRoom = useAuthStore((state) => state.setRoom);
  const [roomJoinCodeInput, setRoomJoinCodeInput] = useState<string>("");
  const user = useAuthStore((state) => state.user);
  const checkAuth = useAuthStore((state) => state.checkAuth);

  useEffect(() => {
    checkAuth();
  }, [checkAuth]);

  useEffect(() => {
    if (!user) {
      navigate("/login");
      return;
    }
  }, [user, navigate]);

  async function handleCreateRoom(e: React.BaseSyntheticEvent) {
    console.log(typeof e);
    e.preventDefault();
    await roomService
      .createRoom()
      .then((data) => {
        if (!data) return;

        const { room_id } = data;

        setRoom(room_id);
        navigate(`/room/${room_id}`);
      })
      .catch((err) => {
        alert(
          err.response?.data?.code || err.message || "Failed to create room",
        );
      });
  }

  async function handleJoinRoom(e: React.BaseSyntheticEvent) {
    e.preventDefault();

    await roomService
      .joinRoomByCode(roomJoinCodeInput)
      .then(({ room_id }) => {
        setRoom(room_id);
        navigate(`/room/${room_id}`);
      })
      .catch((err) => {
        alert(err.response?.data?.code || err.message || "Failed to join room");
      });
  }

  if (user_room_id || !is_authenticated) return;

  return (
    <section className="flex gap-2 flex-col">
      <p>Become host</p>
      <button onClick={handleCreateRoom}>Create Room</button>
      <p>Join with code</p>
      <form className="flex gap-2" onSubmit={handleJoinRoom}>
        <input
          type="text"
          placeholder="ABCDEF"
          value={roomJoinCodeInput}
          pattern="^[a-zA-Z0-9]+$"
          minLength={6}
          maxLength={6}
          onChange={(e) => setRoomJoinCodeInput(e.target.value)}
          required
          className="bg-zinc-900 border border-zinc-700 p-2 rounded uppercase"
        />
        <button type="submit">Join Room</button>
      </form>
    </section>
  );
}
