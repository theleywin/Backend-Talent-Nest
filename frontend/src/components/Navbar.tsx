import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { axiosInstance } from "../lib/axios";
import { Link } from "react-router-dom";
import { removeToken } from "../utils/auth.ts";
import { Bell, Home, LogOut, User, Users } from "lucide-react";
import { getAuthUser } from "../lib/queries";

const Navbar = () => {
    const { data: authUser } = useQuery({
        queryKey: ["authUser"],
        queryFn: getAuthUser,
    });
    const queryClient = useQueryClient();

    const { data: notifications } = useQuery({
        queryKey: ["notifications"],
        queryFn: async () => axiosInstance.get("/notifications"),
        enabled: !!authUser,
    });

    const { data: connectionRequests } = useQuery({
        queryKey: ["connectionRequests"],
        queryFn: async () => axiosInstance.get("/connections/requests"),
        enabled: !!authUser,
    });

    const { mutate: logout } = useMutation({
        mutationFn: () => axiosInstance.post("/auth/logout"),
        onSuccess: () => {
            // Eliminar token del localStorage
            removeToken();
            console.log("Logged out!");
            queryClient.invalidateQueries({ queryKey: ["authUser"] });
        },
    });

    const unreadNotificationCount = notifications?.data ? notifications?.data.filter((notif: { read: any; }) => !notif.read).length : 0;
    const unreadConnectionRequestsCount = connectionRequests?.data?.length;

    return (
        <nav className='bg-white shadow-md sticky top-0 z-10'>
            <div className='max-w-7xl mx-auto px-4'>
                <div className='flex justify-between items-center py-3'>
                    <div className='flex items-center space-x-4'>
                        <Link to='/'>
                            <img className='h-16 rounded' src='/logo.png' alt='TalentNest' />
                        </Link>
                    </div>
                    <div className='flex items-center gap-2 md:gap-6'>
                        {authUser ? (
                            <>
                                <Link to={"/"} className='text-neutral flex flex-col items-center hover:text-green-600'>
                                    <Home size={20} />
                                </Link>
                                <Link to='/network' className='text-neutral flex flex-col items-center relative hover:text-green-600'>
                                    <Users size={20} />
                                    {unreadConnectionRequestsCount > 0 && (
                                        <span
                                            className='absolute -top-1 -right-1 md:right-4 bg-green-900 text-white text-xs
										rounded-full size-3 md:size-4 flex items-center justify-center'
                                        >
											{unreadConnectionRequestsCount}
										</span>
                                    )}
                                </Link>
                                <Link to='/notifications' className='text-neutral flex flex-col items-center relative hover:text-green-600'>
                                    <Bell size={20} />
                                    {unreadNotificationCount > 0 && (
                                        <span
                                            className='absolute -top-1 -right-1 md:right-4 bg-green-900 text-white text-xs
										rounded-full size-3 md:size-4 flex items-center justify-center'
                                        >
											{unreadNotificationCount}
										</span>
                                    )}
                                </Link>
                                <Link
                                    to={`/profile/${authUser.username}`}
                                    className='text-neutral flex flex-col items-center hover:text-green-600'
                                >
                                    <User size={20} />
                                </Link>
                                <button
                                    className='flex items-center space-x-1 text-sm text-gray-600 hover:text-red-700'
                                    onClick={() => logout()}
                                >
                                    <LogOut size={20} />
                                </button>
                            </>
                        ) : (
                            <>
                                <Link to='/login' className='btn btn-ghost text-black hover:text-white'>
                                    Sign In
                                </Link>
                                <Link to='/signup' className='btn btn-neutral'>
                                    Join us!
                                </Link>
                            </>
                        )}
                    </div>
                </div>
            </div>
        </nav>
    );
};
export default Navbar;