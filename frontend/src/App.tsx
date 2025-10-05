import Layout from "./components/Layout.tsx";
import SignupPage from "./pages/auth/SignupPage.tsx";
import HomePage from "./pages/HomePage.tsx";
import LoginPage from "./pages/auth/LoginPage.tsx";
import NoMatchPage from "./pages/NoMatchPage.tsx";
import {Navigate, Route, Routes} from "react-router-dom";
import {Toaster} from "react-hot-toast";
import {useQuery} from "@tanstack/react-query";
import NotificationPage from "./pages/NotificationPage.tsx";
import NetworkPage from "./pages/NetworkPage.tsx";
import PostPage from "./pages/PostPage.tsx";
import ProfilePage from "./pages/ProfilePage.tsx";
import { getAuthUser } from "./lib/queries";


function App() {

    const { data: authUser } = useQuery({
        queryKey: ["authUser"],
        queryFn: getAuthUser,
    });

    return (
      <Layout>
        <Routes>
            <Route path='/' element={authUser ? <HomePage /> : <Navigate to={"/login"} />} />
            <Route path='/signup' element={!authUser ? <SignupPage /> : <Navigate to={"/"} />} />
            <Route path="/login" element={!authUser ? <LoginPage /> : <Navigate to={"/"} />} />
            <Route path="/notifications" element={authUser ? <NotificationPage /> : <Navigate to={"/login"} />} />
            <Route path="/network" element={authUser ? <NetworkPage /> : <Navigate to={"/login"} />} />
            <Route path='/post/:postId' element={authUser ? <PostPage /> : <Navigate to={"/login"} />} />
            <Route path='/profile/:username' element={authUser ? <ProfilePage /> : <Navigate to={"/login"} />} />
            <Route path="*" element={<NoMatchPage />} />
        </Routes>
          <Toaster/>
      </Layout>
  )};

export default App
