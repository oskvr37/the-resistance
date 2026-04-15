import { useAuthStore } from "@/store/auth";
import { AuthMeData } from "@/api/auth";
import Avatar from "@/components/Avatar";
import { useNavigate } from "react-router";

export default function User({ user }: { user: AuthMeData | null }) {
  const logout = useAuthStore((state) => state.logout);
  const navigate = useNavigate();

  if (!user) return;

  return (
    <section className="w-full rounded border border-zinc-600 p-4 space-y-2 flex items-center justify-between">
      <div className="space-y-4">
        <div className="flex justify-between items-center">
          <p>Hello, {user.username}!</p>
          <button
            onClick={() => {
              logout();
              navigate("/login");
            }}
          >
            Logout
          </button>
        </div>
        <div className="text-sm text-zinc-500 font-light font-mono">
          <p>uid {user.id}</p>
          {user.room_id && <p>room {user.room_id}</p>}
        </div>
      </div>
      <Avatar seed={user.avatar} options={{}} />
    </section>
  );
} 
