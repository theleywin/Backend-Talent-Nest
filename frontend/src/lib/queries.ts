import { axiosInstance } from "./axios";
import { toast } from "react-hot-toast";

export const getAuthUser = async () => {
    try {
        const res = await axiosInstance.get("/auth/me");
        console.log("éxito en obtener token" + res.data);
        return res.data;
    } catch (err: any) {
        if (err?.response?.status === 401) {
            return null;
        }
        toast.error(err?.response?.data?.message || "Something went wrong");
        throw err;
    }
};

