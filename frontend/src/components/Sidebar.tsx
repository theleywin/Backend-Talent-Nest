import { Link } from "react-router-dom";
import { Home, UserPlus, Bell } from "lucide-react";

export default function Sidebar({ user }) {
    return (
        <div className='bg-gray-100 rounded-lg shadow'>
            <div className='p-4 text-center'>
                <div
                    className='h-16 rounded-t-lg bg-cover bg-center'
                    style={{
                        backgroundImage: `url("${user.bannerImg || "/banner.png"}")`,
                    }}
                />
                <Link to={`/profile/${user.username}`}>
                    <img
                        src={user.profilePicture || "/avatar.png"}
                        alt={user.name}
                        className='w-20 h-20 rounded-full mx-auto mt-[-40px]'
                    />
                    <h2 className='text-xl text-black font-semibold mt-2'>{user.name}</h2>
                </Link>
                <p className='text-gray-800'>{user.headline}</p>
                <p className='text-gray-800 text-xs'>{user.connections.length} connections</p>
            </div>
            <div className='border-t border-gray-800 p-4'>
                <nav>
                    <ul className='space-y-2'>
                        <li>
                            <Link
                                to='/'
                                className='flex text-black items-center py-2 px-4 rounded-md hover:text-green-700 transition-colors'
                            >
                                <Home className='mr-2' size={20} /> Home
                            </Link>
                        </li>
                        <li>
                            <Link
                                to='/network'
                                className='flex text-black items-center py-2 px-4 rounded-md hover:text-green-700 transition-colors'
                            >
                                <UserPlus className='mr-2' size={20} /> My Network
                            </Link>
                        </li>
                        <li>
                            <Link
                                to='/notifications'
                                className='flex text-black items-center py-2 px-4 rounded-md hover:text-green-700 transition-colors'
                            >
                                <Bell className='mr-2' size={20} /> Notifications
                            </Link>
                        </li>
                    </ul>
                </nav>
            </div>
            <div className='border-t border-black p-4'>
                <Link to={`/profile/${user.username}`} className='text-sm text-black hover:text-green-700 font-semibold'>
                    Visit your profile
                </Link>
            </div>
        </div>
    );
}