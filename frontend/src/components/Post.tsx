import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { axiosInstance } from "../lib/axios";
import toast from "react-hot-toast";
import { Link, useParams } from "react-router-dom";
import { Loader, MessageCircle, Send, Repeat2, ThumbsUp, Trash2, X } from "lucide-react";
import { formatDistanceToNow } from "date-fns";
import { getAuthUser } from "../lib/queries";

import PostAction from "./PostAction";

interface User {
    _id: string;
    name: string;
    username: string;
    headline: string;
    profilePicture: string;
}

interface Comment {
    _id: string;
    content: string;
    user: User;
    createdAt: string;
}

interface Post {
    _id: string;
    author: User;
    content: string;
    image?: string;
    likes: string[];
    comments: Comment[];
    createdAt: string;
    repost?: Post;
}

interface PostProps {
    post: Post;
}

interface ApiError {
    response?: {
        data?: {
            message?: string;
        };
    };
    message: string;
}

const Post = ({ post }: PostProps) => {
    const { postId } = useParams();

    const {data: authUser} = useQuery({
        queryKey: ["authUser"],
        queryFn: getAuthUser,
    });
    const [showComments, setShowComments] = useState(false);
    const [newComment, setNewComment] = useState("");
    const [comments, setComments] = useState<Comment[]>(post.comments || []);
    const [showRepostModal, setShowRepostModal] = useState(false);
    const [repostComment, setRepostComment] = useState("");

    // SOLUCIÓN SIMPLE: Comparar directamente los IDs como strings
    const isOwner = !!(authUser && post.author && String(authUser._id) === String(post.author._id));

    // Arreglar isLiked: post.likes viene como array de UserDto
    const isLiked = authUser && Array.isArray(post.likes)
        ? post.likes.some((like: any) => {
            const likeId = typeof like === 'string' ? like : like._id;
            return String(likeId) === String(authUser._id);
          })
        : false;

    console.log('🔍 Debug - isOwner:', isOwner, 'authUser._id:', authUser?._id, 'post.author._id:', post.author?._id);

    const queryClient = useQueryClient();

    const { mutate: deletePost, isPending: isDeletingPost } = useMutation({
        mutationFn: async () => {
            await axiosInstance.delete(`/posts/delete/${post._id}`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["posts"] });
            toast.success("Post deleted successfully");
        },
        onError: (error: ApiError) => {
            toast.error(error?.response?.data?.message || error.message || "Failed to delete post");
        },
    });

    const { mutate: createComment, isPending: isAddingComment } = useMutation({
        mutationFn: async (newComment: string) => {
            await axiosInstance.post(`/posts/${post._id}/comment`, { content: newComment });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["posts"] });
            toast.success("Comment added successfully");
        },
        onError: (err: ApiError) => {
            toast.error(err?.response?.data?.message || "Failed to add comment");
        },
    });

    const { mutate: likePost, isPending: isLikingPost } = useMutation({
        mutationFn: async () => {
            await axiosInstance.post(`/posts/${post._id}/like`);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["posts"] });
            queryClient.invalidateQueries({ queryKey: ["post", postId] });
        },
    });

    const { mutate: repostPost, isPending: isReposting } = useMutation({
        mutationFn: async (content: string) => {
            await axiosInstance.post("/posts/create", {
                content: content,
                repost: post.repost ? post.repost._id : post._id
            });
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: ["posts"] });
            toast.success("Post reposted successfully");
            setShowRepostModal(false);
            setRepostComment("");
        },
        onError: (err: ApiError) => {
            toast.error(err?.response?.data?.message || "Failed to repost");
        },
    });

    const handleDeletePost = () => {
        if (!window.confirm("Are you sure you want to delete this post?")) return;
        deletePost();
    };

    const handleLikePost = async () => {
        if (isLikingPost) return;
        likePost();
    };

    const handleRepost = async () => {
        if (!authUser) {
            toast.error("You must be logged in to repost.");
            return;
        }
        setShowRepostModal(true);
    };

    const handleRepostSubmit = () => {
        if (isReposting) return;
        repostPost(repostComment);
    };

    const handleRepostCancel = () => {
        setShowRepostModal(false);
        setRepostComment("");
    };

    const handleAddComment = async (e: React.FormEvent) => {
        e.preventDefault();
        if (!authUser) {
            toast.error("You must be logged in to comment.");
            return;
        }
        if (newComment.trim()) {
            createComment(newComment);
            setNewComment("");
            setComments([
                ...comments,
                {
                    _id: Date.now().toString(),
                    content: newComment,
                    user: {
                        _id: authUser._id,
                        name: authUser.name,
                        username: authUser.username,
                        headline: authUser.headline,
                        profilePicture: authUser.profilePicture,
                    },
                    createdAt: new Date().toISOString(),
                },
            ]);
        }
    };

    return (
        <div className='bg-gray-200 rounded-lg shadow mb-4'>
            {/* Repost Modal */}
            {showRepostModal && (
                <div className='fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50'>
                    <div className='bg-white rounded-lg p-6 max-w-lg w-full mx-4'>
                        <div className='flex justify-between items-center mb-4'>
                            <h3 className='text-xl font-semibold text-black'>Repost</h3>
                            <button onClick={handleRepostCancel} className='text-gray-500 hover:text-gray-700'>
                                <X size={24} />
                            </button>
                        </div>

                        <div className='mb-4'>
                            <textarea
                                className='w-full text-black p-3 rounded-lg bg-gray-100 focus:bg-white focus:outline-none resize-none border border-gray-300'
                                placeholder='Add a comment to your repost (optional)...'
                                value={repostComment}
                                onChange={(e) => setRepostComment(e.target.value)}
                                rows={3}
                            />
                        </div>

                        {/* Preview of the post being reposted */}
                        <div className='border border-gray-300 rounded-lg p-3 mb-4 bg-gray-50'>
                            <div className='flex items-center mb-2'>
                                <img
                                    src={(post.repost?.author?.profilePicture || post.author.profilePicture) || "/avatar.png"}
                                    alt={(post.repost?.author?.name || post.author.name)}
                                    className='size-8 rounded-full mr-2'
                                />
                                <div>
                                    <h4 className='font-semibold text-sm text-black'>
                                        {post.repost?.author?.name || post.author.name}
                                    </h4>
                                    <p className='text-xs text-gray-500'>
                                        {post.repost?.author?.headline || post.author.headline}
                                    </p>
                                </div>
                            </div>
                            <p className='text-sm text-black'>{post.repost?.content || post.content}</p>
                            {(post.repost?.image || post.image) && (
                                <img
                                    src={post.repost?.image || post.image}
                                    alt='Post content'
                                    className='rounded-lg w-full mt-2 max-h-48 object-cover'
                                />
                            )}
                        </div>

                        <div className='flex justify-end gap-3'>
                            <button
                                onClick={handleRepostCancel}
                                className='px-4 py-2 text-gray-700 bg-gray-200 rounded-lg hover:bg-gray-300 transition-colors'
                                disabled={isReposting}
                            >
                                Cancel
                            </button>
                            <button
                                onClick={handleRepostSubmit}
                                className='px-4 py-2 bg-green-900 text-white rounded-lg hover:bg-green-800 transition-colors flex items-center gap-2'
                                disabled={isReposting}
                            >
                                {isReposting ? (
                                    <>
                                        <Loader size={18} className='animate-spin' />
                                        Reposting...
                                    </>
                                ) : (
                                    <>
                                        <Repeat2 size={18} />
                                        Repost
                                    </>
                                )}
                            </button>
                        </div>
                    </div>
                </div>
            )}

            <div className='p-4'>
                <div className='flex items-center text-black justify-between mb-4'>
                    <div className='flex items-center'>
                        <Link to={`/profile/${post?.author?.username}`}>
                            <img
                                src={post.author.profilePicture || "/avatar.png"}
                                alt={post.author.name}
                                className='size-10 rounded-full mr-3'
                            />
                        </Link>

                        <div>
                            <Link to={`/profile/${post?.author?.username}`}>
                                <h3 className='font-semibold'>{post.author.name}</h3>
                            </Link>
                            <p className='text-xs text-green-800'>{post.author.headline}</p>
                            <p className='text-xs text-green-800'>
                                {formatDistanceToNow(new Date(post.createdAt), { addSuffix: true })}
                            </p>
                        </div>
                    </div>
                    {isOwner && (
                        <button onClick={handleDeletePost} className='text-red-500 hover:text-red-700'>
                            {isDeletingPost ? <Loader size={18} className='animate-spin' /> : <Trash2 size={18} />}
                        </button>
                    )}
                </div>
                <p className='mb-4 text-black'>{post.content}</p>
                {post.image && <img src={post.image} alt='Post content' className='rounded-lg w-full mb-4' />}

                {post.repost && (
                    <div className='border border-gray-300 rounded-lg p-3 mb-4 bg-white'>
                        <div className='flex items-center mb-2'>
                            <Link to={`/profile/${post.repost.author?.username}`}>
                                <img
                                    src={post.repost.author.profilePicture || "/avatar.png"}
                                    alt={post.repost.author.name}
                                    className='size-8 rounded-full mr-2'
                                />
                            </Link>
                            <div>
                                <Link to={`/profile/${post.repost.author?.username}`}>
                                    <h4 className='font-semibold text-sm'>{post.repost.author.name}</h4>
                                </Link>
                                <p className='text-xs text-gray-500'>{post.repost.author.headline}</p>
                            </div>
                        </div>
                        <p className='text-sm text-black'>{post.repost.content}</p>
                        {post.repost.image && (
                            <img src={post.repost.image} alt='Reposted content' className='rounded-lg w-full mt-2' />
                        )}
                    </div>
                )}

                <div className='flex justify-between text-green-800'>
                    <PostAction
                        icon={<ThumbsUp size={18} className={isLiked ? "text-green-700  fill-green-400" : ""} />}
                        text={`Like (${Array.isArray(post.likes) ? post.likes.length : 0})`}
                        onClick={handleLikePost}
                    />

                    <PostAction
                        icon={<MessageCircle size={18} />}
                        text={`Comment (${comments.length})`}
                        onClick={() => setShowComments(!showComments)}
                    />
                    <PostAction
                        icon={isReposting ? <Loader size={18} className='animate-spin' /> : <Repeat2 size={18} />}
                        text='Repost'
                        onClick={handleRepost}
                    />
                </div>
            </div>

            {showComments && (
                <div className='px-4 pb-4'>
                    <div className='mb-4 max-h-60 overflow-y-auto'>
                        {comments.map((comment: Comment) => (
                            <div key={comment._id} className='mb-2 bg-white text-black p-2 rounded flex items-start'>
                                <img
                                    src={comment.user.profilePicture || "/avatar.png"}
                                    alt={comment.user.name}
                                    className='w-8 h-8 rounded-full mr-2 flex-shrink-0'
                                />
                                <div className='flex-grow'>
                                    <div className='flex items-center mb-1'>
                                        <span className='font-semibold mr-2'>{comment.user.name}</span>
                                        <span className='text-xs text-green-800'>
                                            {formatDistanceToNow(new Date(comment.createdAt))}
                                        </span>
                                    </div>
                                    <p>{comment.content}</p>
                                </div>
                            </div>
                        ))}
                    </div>

                    <form onSubmit={handleAddComment} className='flex text-black focus:'>
                        <input
                            type='text'
                            value={newComment}
                            onChange={(e) => setNewComment(e.target.value)}
                            placeholder='Add a comment...'
                            className='flex-grow p-2.5 rounded-l-full bg-gray-100'
                        />

                        <button
                            type='submit'
                            className='bg-green-900 text-white hover:bg-gray-100 hover:text-green-900 p-2.5 rounded-r-full hover:bg-primary-dark transition duration-300'
                            disabled={isAddingComment}
                        >
                            {isAddingComment ? <Loader size={18} className='animate-spin' /> : <Send size={22} />}
                        </button>
                    </form>
                </div>
            )}
        </div>
    );
};
export default Post;
