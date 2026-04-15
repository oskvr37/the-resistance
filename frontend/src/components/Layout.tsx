import { Outlet } from "react-router";
import User from "./User";
import { useAuthStore } from "@/store/auth";

export default function Layout() {
  const user = useAuthStore((state) => state.user);

  return (
    <main>
      <User user={user} />
      <Outlet />
    </main>
  );
}
