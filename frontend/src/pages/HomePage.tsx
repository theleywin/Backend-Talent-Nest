import { useQuery } from "@tanstack/react-query";
import { axiosInstance } from "../lib/axios";
import Sidebar from "../components/Sidebar";
import { Users } from "lucide-react";
import PostCreation from "../components/PostCreation.tsx";
import Post from "../components/Post.tsx";
import RecommendedUser from "../components/RecommendedUser.tsx";
import UserSearch from "../components/UserSearch.tsx";
import { getAuthUser } from "../lib/queries";

const HomePage = () => {
    const {data: authUser} = useQuery({
        queryKey: ["authUser"],
        queryFn: getAuthUser,
    });

    const {data: recommendedUsers} = useQuery({
        queryKey: ["recommendedUsers"],
        queryFn: async () => {
            const res = await axiosInstance.get("/users/suggestions");
            return res.data;
        },
    });

    const {data: posts} = useQuery({
        queryKey: ["posts"],
        queryFn: async () => {
            const res = await axiosInstance.get("/posts");
            return res.data;
        },
    });

    return (
        <div className='grid grid-cols-1 lg:grid-cols-4 gap-6'>
            <div className='hidden lg:block lg:col-span-1'>
                <Sidebar user={authUser} />
            </div>

            <div className='col-span-1 lg:col-span-2 order-first lg:order-none'>
                <UserSearch />
                <PostCreation user={authUser} />

                {posts?.map((post: any) => (
                    <Post key={post._id} post={post} />
                ))}

                {posts?.length === 0 && (
                    <div className='bg-white rounded-lg shadow p-8 text-center'>
                        <div className='mb-6'>
                            <Users size={64} className='mx-auto text-green-700' />
                        </div>
                        <h2 className='text-2xl font-bold mb-4 text-gray-800'>No Posts Yet</h2>
                        <p className='text-gray-600 mb-6'>Connect with others to start seeing posts in your feed!</p>
                    </div>
                )}
            </div>
            {recommendedUsers?.length > 0 && (
                <div className='col-span-1 lg:col-span-1 hidden lg:block'>
                    <div className='bg-gray-100 rounded-lg shadow p-4 text-green-800'>
                        <h2 className='font-semibold mb-4'>People you may know</h2>
                        {recommendedUsers?.map((user: any) => (
                            <RecommendedUser key={user._id} user={user} />
                        ))}
                    </div>
                </div>
            )}
        </div>
    );
}
export default HomePage
