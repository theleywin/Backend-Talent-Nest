import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { axiosInstance } from "../lib/axios";
import { toast } from "react-hot-toast";
import { ExternalLink, Eye, MessageSquare, ThumbsUp, Trash2, UserPlus } from "lucide-react";
import { Link } from "react-router-dom";
import Sidebar from "../components/Sidebar";
import { formatDistanceToNow } from "date-fns";
import {getAuthUser} from "../lib/queries.ts";


// Type for Notification
interface Notification {
    _id: string;
    type: string;
    relatedUser: {
        username: string;
        name: string;
        profilePicture?: string;
    };
    relatedPost?: {
        _id: string;
        image?: string;
        content?: string;
    } | null;
    read: boolean;
    createdAt: string;
    [key: string]: any;
}

const extractId = (raw: any): string | undefined => {
    if (!raw) return undefined;
    if (typeof raw === "string") return raw;
    if (raw.$oid && typeof raw.$oid === "string") return raw.$oid;
    if (raw.id && typeof raw.id === "string") return raw.id;
    try {
        return String(raw);
    } catch {
        return undefined;
    }
}

const NotificationsPage = () => {
    const { data: authUser } = useQuery({
        queryKey: ["authUser"],
        queryFn: getAuthUser,
    });

    const queryClient = useQueryClient();

    const { data: notifications, isLoading } = useQuery<Notification[]>({
        queryKey: ["notifications"],
        queryFn: async () => {
            const res = await axiosInstance.get("/notifications");
            const rawList = res.data?.data ?? res.data ?? [];
            return (rawList || []).map((n: any) => ({
                _id: extractId(n._id ?? n.id) ?? crypto.randomUUID(),
                type: n.type,
                relatedUser: n.relatedUser,
                relatedPost: n.relatedPost ?? null,
                createdAt: n.createdAt,
                read: n.read ?? false,
                ...n,
            } as Notification));
        },
    });

    const { mutate: markAsReadMutation } = useMutation<void, unknown, string>({
        mutationFn: (id: string) => axiosInstance.put(`/notifications/${id}/read`),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["notifications"] });
        },
    });

    const { mutate: deleteNotificationMutation } = useMutation<void, unknown, string>({
        mutationFn: (id: string) => axiosInstance.delete(`/notifications/${id}`),
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["notifications"] });
            toast.success("Notification deleted");
        },
    });

    const renderNotificationIcon = (type: string): JSX.Element | null => {
        switch (type) {
            case "like":
                return <ThumbsUp className='text-blue-500' />;
            case "comment":
                return <MessageSquare className='text-green-500' />;
            case "connectionAccepted":
                return <UserPlus className='text-purple-500' />;
            default:
                return null;
        }
    };

    const renderNotificationContent = (notification: Notification): JSX.Element | null => {
        switch (notification.type) {
            case "like":
                return (
                    <span>
                        <strong>{notification.relatedUser.name}</strong> liked your post
                    </span>
                );
            case "comment":
                return (
                    <span>
                        <Link to={`/profile/${notification.relatedUser.username}`} className='font-bold'>
                            {notification.relatedUser.name}
                        </Link>{" "}
                        commented on your post
                    </span>
                );
            case "connectionAccepted":
                return (
                    <span>
                        <Link to={`/profile/${notification.relatedUser.username}`} className='font-bold'>
                            {notification.relatedUser.name}
                        </Link>{" "}
                        accepted your connection request
                    </span>
                );
            default:
                return null;
        }
    };

    const renderRelatedPost = (relatedPost: Notification["relatedPost"]): JSX.Element | null => {
        if (!relatedPost) return null;

        return (
            <Link
                to={`/post/${relatedPost._id}`}
                className='mt-2 p-2 bg-gray-50 rounded-md flex items-center space-x-2 hover:bg-gray-100 transition-colors'
            >
                {relatedPost.image && (
                    <img src={relatedPost.image} alt='Post preview' className='w-10 h-10 object-cover rounded' />
                )}
                <div className='flex-1 overflow-hidden'>
                    <p className='text-sm text-gray-600 truncate'>{relatedPost.content}</p>
                </div>
                <ExternalLink size={14} className='text-gray-400' />
            </Link>
        );
    };

    return (
        <div className='grid grid-cols-1 lg:grid-cols-4 gap-6'>
            <div className='col-span-1 lg:col-span-1'>
                <Sidebar user={authUser} />
            </div>
            <div className='col-span-1 lg:col-span-3'>
                <div className='bg-white rounded-lg shadow p-6'>
                    <h1 className='text-2xl font-bold mb-6'>Notifications</h1>

                    {isLoading ? (
                        <p>Loading notifications...</p>
                    ) : notifications && notifications.length > 0 ? (
                        <ul>
                            {notifications.map((notification: Notification) => (
                                <li
                                    key={notification._id}
                                    className={`bg-white border rounded-lg p-4 my-4 transition-all hover:shadow-md ${
                                        !notification.read ? "border-blue-500" : "border-gray-200"
                                    }`}
                                >
                                    <div className='flex items-start justify-between'>
                                        <div className='flex items-center space-x-4'>
                                            <Link to={`/profile/${notification.relatedUser.username}`}>
                                                <img
                                                    src={notification.relatedUser.profilePicture || "/avatar.png"}
                                                    alt={notification.relatedUser.name}
                                                    className='w-12 h-12 rounded-full object-cover'
                                                />
                                            </Link>

                                            <div>
                                                <div className='flex items-center gap-2'>
                                                    <div className='p-1 bg-gray-100 rounded-full'>
                                                        {renderNotificationIcon(notification.type)}
                                                    </div>
                                                    <p className='text-sm'>{renderNotificationContent(notification)}</p>
                                                </div>
                                                <p className='text-xs text-gray-500 mt-1'>
                                                    {formatDistanceToNow(new Date(notification.createdAt), {
                                                        addSuffix: true,
                                                    })}
                                                </p>
                                                {renderRelatedPost(notification.relatedPost)}
                                            </div>
                                        </div>

                                        <div className='flex gap-2'>
                                            {!notification.read && (
                                                <button
                                                    onClick={() => markAsReadMutation(notification._id)}
                                                    className='p-1 bg-blue-100 text-blue-600 rounded hover:bg-blue-200 transition-colors'
                                                    aria-label='Mark as read'
                                                >
                                                    <Eye size={16} />
                                                </button>
                                            )}

                                            <button
                                                onClick={() => deleteNotificationMutation(notification._id)}
                                                className='p-1 bg-red-100 text-red-600 rounded hover:bg-red-200 transition-colors'
                                                aria-label='Delete notification'
                                            >
                                                <Trash2 size={16} />
                                            </button>
                                        </div>
                                    </div>
                                </li>
                            ))}
                        </ul>
                    ) : (
                        <p>No notification at the moment.</p>
                    )}
                </div>
            </div>
        </div>
    );
};
export default NotificationsPage;