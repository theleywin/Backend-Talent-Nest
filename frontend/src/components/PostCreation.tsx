import { useMutation, useQueryClient } from "@tanstack/react-query";
import { useState } from "react";
import { axiosInstance } from "../lib/axios";
import toast from "react-hot-toast";
import { Image, Loader } from "lucide-react";

const PostCreation = ({ user }) => {
    const [content, setContent] = useState("");
    const [image, setImage] = useState(null);
    const [imagePreview, setImagePreview] = useState(null);

    const queryClient = useQueryClient();

    const { mutate: createPostMutation, isPending } = useMutation({
        mutationFn: async (postData) => {
            const res = await axiosInstance.post("/posts/create", postData, {
                headers: { "Content-Type": "application/json" }
            });
            return res.data;
        },
        onSuccess: () => {
            resetForm();
            toast.success("Post created successfully");
            queryClient.invalidateQueries({ queryKey: ["posts"] });
        },
        onError: (err) => {
            toast.error(err.response.data.message || "Failed to create post");
        },
    });

    const handlePostCreation = async () => {
        try {
            const postData = { content };
            if (image) postData.image = await readFileAsDataURL(image);

            createPostMutation(postData);
        } catch (error) {
            console.error("Error in handlePostCreation:", error);
        }
    };

    const resetForm = () => {
        setContent("");
        setImage(null);
        setImagePreview(null);
    };

    const handleImageChange = (e) => {
        const file = e.target.files[0];
        setImage(file);
        if (file) {
            readFileAsDataURL(file).then(setImagePreview);
        } else {
            setImagePreview(null);
        }
    };

    const readFileAsDataURL = (file) => {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();
            reader.onloadend = () => resolve(reader.result);
            reader.onerror = reject;
            reader.readAsDataURL(file);
        });
    };

    return (
        <div className='bg-gray-50 rounded-lg shadow mb-4 p-4'>
            <div className='flex space-x-3'>
                <img src={user.profilePicture || "/avatar.png"} alt={user.name} className='size-12 rounded-full' />
                <textarea
                    placeholder="What's on your mind?"
                    className='w-full text-black p-3 rounded-lg bg-gray-100 hover:bg-gray-50 focus:bg-gray-100 focus:outline-none resize-none transition-colors duration-200 min-h-[100px]'
                    value={content}
                    onChange={(e) => setContent(e.target.value)}
                />
            </div>

            {imagePreview && (
                <div className='mt-4'>
                    <img src={imagePreview} alt='Selected' className='w-full h-auto rounded-lg' />
                </div>
            )}

            <div className='flex justify-between items-center mt-4'>
                <div className='flex space-x-4'>
                    <label className='flex items-center text-black hover:text-green-700 transition-colors duration-200 cursor-pointer'>
                        <Image size={20} className='mr-2' />
                        <span>Photo</span>
                        <input type='file' accept='image/*' className='hidden' onChange={handleImageChange} />
                    </label>
                </div>

                <button
                    className='bg-gray-800 text-white border-gray-800 rounded-lg px-4 py-2 hover:bg-green-700 transition-colors duration-200'
                    onClick={handlePostCreation}
                    disabled={isPending}
                >
                    {isPending ? <Loader className='size-5 animate-spin' /> : "Share"}
                </button>
            </div>
        </div>
    );
};
export default PostCreation;