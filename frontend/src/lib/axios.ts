import axios from "axios";
import { getToken } from "../utils/auth.ts";

export const axiosInstance = axios.create({
    baseURL: "http://backend-service:3000/api/v1",
});

// Interceptor para agregar el token JWT a cada request
axiosInstance.interceptors.request.use(
    (config) => {
        const token = getToken();
        if (token) {
            config.headers.Authorization = `Bearer ${token}`;
        }
        return config;
    },
    (error) => {
        return Promise.reject(error);
    }
);