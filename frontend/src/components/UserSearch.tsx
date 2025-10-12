import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { axiosInstance } from "../lib/axios";
import { Search } from "lucide-react";
import { Link } from "react-router-dom";

const UserSearch = () => {
    const [searchTerm, setSearchTerm] = useState("");

    const { data: users, isLoading } = useQuery({
        queryKey: ["searchUsers", searchTerm],
        queryFn: async () => {
            if (!searchTerm.trim()) return [];
            const res = await axiosInstance.get(`/users/search?query=${searchTerm}`);
            return res.data;
        },
        enabled: searchTerm.length > 0,
    });

    return (
        <div className="bg-white rounded-lg shadow p-4 mb-6">
            <div className="relative">
                <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-gray-400" size={20} />
                <input
                    type="text"
                    placeholder="Buscar usuarios..."
                    value={searchTerm}
                    onChange={(e) => setSearchTerm(e.target.value)}
                    className="w-full pl-10 pr-4 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500"
                />
            </div>

            {searchTerm && (
                <div className="mt-4 max-h-96 overflow-y-auto">
                    {isLoading && <p className="text-gray-500 text-center py-2">Buscando...</p>}
                    
                    {users && users.length === 0 && !isLoading && (
                        <p className="text-gray-500 text-center py-2">No se encontraron usuarios</p>
                    )}

                    {users?.map((user: any) => (
                        <Link
                            key={user._id}
                            to={`/profile/${user.username}`}
                            className="flex items-center gap-3 p-3 hover:bg-gray-50 rounded-lg transition"
                            onClick={() => setSearchTerm("")}
                        >
                            <img
                                src={user.profilePicture || "/avatar.png"}
                                alt={user.name}
                                className="w-10 h-10 rounded-full object-cover"
                            />
                            <div>
                                <p className="font-semibold text-gray-800">{user.name}</p>
                                <p className="text-sm text-gray-500">@{user.username}</p>
                                {user.headline && (
                                    <p className="text-xs text-gray-400 truncate">{user.headline}</p>
                                )}
                            </div>
                        </Link>
                    ))}
                </div>
            )}
        </div>
    );
};

export default UserSearch;

