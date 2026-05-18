import axios from "axios";
import type { AxiosRequestConfig } from "axios";

const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 10000,
});

request.interceptors.response.use(
  (res) => res.data,
  (error) => {
    if (error.response?.status === 401) {
      window.location.href = "/login";
    }
    return Promise.reject(error);
  },
);

export default request;

export const get = <T>(url: string, config?: AxiosRequestConfig) =>
  request.get<unknown, T>(url, config);
export const post = <T>(
  url: string,
  data?: unknown,
  config?: AxiosRequestConfig,
) => request.post<unknown, T>(url, data, config);
export const put = <T>(
  url: string,
  data?: unknown,
  config?: AxiosRequestConfig,
) => request.put<unknown, T>(url, data, config);
export const del = <T>(url: string, config?: AxiosRequestConfig) =>
  request.delete<unknown, T>(url, config);
