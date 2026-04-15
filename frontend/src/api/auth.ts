import {api, ApiResponse} from "@/api";

export interface LoginPayload {
  username: string;
  avatar: string;
}

interface LoginData {
  id: string;
}

export interface AuthMeData {
  id: string;
  avatar: string;
  room_id: string | null;
  username: string;
}

export const authService = {
  getMe: async () => {
    const { data } = await api.get<ApiResponse<AuthMeData>>("/auth/me");
    return data.data;
  },

  login: async (payload: LoginPayload) => {
    const { data } = await api.post<ApiResponse<LoginData>>("/auth", payload);
    return data.data;
  },
};
