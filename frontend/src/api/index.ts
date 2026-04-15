import axios from "axios";

export const BASE_URL = import.meta.env.DEV ? "http://localhost:8000" : "/api";

export const api = axios.create({
  baseURL: BASE_URL,
  withCredentials: true,
});

api.interceptors.response.use(
  (response) => response,
  (error) => {
    switch (error.response?.status) {
      case 401:
        // session expired
        console.warn("Session expired");
        break;

      case 409:
        // unhandled conflict
        console.error(error);
        break;

      default:
        break;
    }
    return Promise.reject(error);
  },
);

export interface ApiResponse<T = object> {
  code: string;
  data: T;
}
