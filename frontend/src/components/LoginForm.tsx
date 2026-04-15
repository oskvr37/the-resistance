import { useEffect, useState } from "react";
import { useAuthStore } from "@/store/auth";
import Avatar from "./Avatar";
import { useNavigate } from "react-router";

const generateRandomHash = (length: number = 16) => {
  const chars =
    "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789";
  return Array.from({ length }, () =>
    chars.charAt(Math.floor(Math.random() * chars.length)),
  ).join("");
};

export default function LoginForm() {
  const login = useAuthStore((state) => state.login);
  const user = useAuthStore((state) => state.user);
  const is_authenticated = useAuthStore((state) => state.is_authenticated);
  const navigate = useNavigate();

  // TODO get room id from url query to navigate after login

  const [username, setUsername] = useState(user?.username || "");
  const [avatar, setAvatar] = useState(user?.avatar || generateRandomHash(16));
  const [isSubmitting, setIsSubmitting] = useState(false);

  useEffect(() => {
    if (user && is_authenticated) {
      navigate("/");
      return;
    }
  });

  const randomizeAvatar = () => {
    setAvatar(generateRandomHash());
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setIsSubmitting(true);

    await login({ username, avatar })
      .then(() => {
        navigate("/");
      })
      .catch((err) => {
        console.error(err);
        alert("Login failed. Check your credentials.");
      })
      .finally(() => {
        setIsSubmitting(false);
      });
  };

  if (user && is_authenticated) {
    return;
  }

  return (
    <form onSubmit={handleSubmit} className="flex flex-col gap-4">
      <h2>Please login!</h2>
      <div className="flex flex-col items-center gap-2">
        <p className="text-zinc-400 text-sm">Click to change avatar!</p>
        <div
          onClick={randomizeAvatar}
          className="cursor-pointer transition-opacity animate-pulse hover:animate-none border-zinc-600 border rounded aspect-square"
        >
          <Avatar seed={avatar} options={{}} />
        </div>
      </div>

      <p className="text-zinc-400 text-sm">Input username! (3+ characters)</p>

      <div className="flex gap-2">
        <input
          type="text"
          className="bg-zinc-900 border border-zinc-700 p-2 rounded"
          placeholder="Username"
          value={username}
          minLength={3}
          required
          pattern="^[a-zA-Z0-9]{3,}$"
          onChange={(e) => setUsername(e.target.value)}
        />

        <input type="text" value={avatar} readOnly className="hidden" />

        <button type="submit" disabled={isSubmitting} className="bg-zinc-800">
          {isSubmitting ? "..." : "JOIN"}
        </button>
      </div>
    </form>
  );
}
