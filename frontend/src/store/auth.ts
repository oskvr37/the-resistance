import { create } from "zustand";
import { persist, createJSONStorage } from "zustand/middleware";
import { authService, LoginPayload, AuthMeData } from "@/api/auth";

interface AuthState {
  is_authenticated: boolean;
  user: AuthMeData | null;
  login: (user_data: LoginPayload) => Promise<void>;
  checkAuth: () => Promise<void>;
  logout: () => void;
  setRoom: (room_id: string | null) => void;
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set) => ({
      is_authenticated: false,
      user: null,

      checkAuth: async () => {
        await authService
          .getMe()
          .then((user) => {
            set({ user, is_authenticated: true });
          })
          .catch((err) => {
            set({ is_authenticated: false });
            console.error("Session expired, but keeping local profile data", {
              err,
            });
          });
      },

      login: async (login_payload) => {
        await authService.login(login_payload).then(({ id }) => {
          set({
            user: { id, ...login_payload, room_id: null },
            is_authenticated: true,
          });
        });
      },

      logout: () => set({ user: null, is_authenticated: false }),

      setRoom: (roomId) =>
        set((state) => ({
          user: state.user ? { ...state.user, room_id: roomId } : null,
        })),
    }),
    {
      name: "resistance-auth-storage",
      storage: createJSONStorage(() => localStorage),
    },
  ),
);
