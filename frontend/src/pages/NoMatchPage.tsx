import {Link} from "react-router-dom";
import {ArrowLeft} from "lucide-react";

const NoMatchPage = () => {

    return (
        <div className="flex min-h-screen flex-col items-center justify-center bg-gray-50 px-6 py-16">
            <div className="mx-auto flex max-w-2xl flex-col items-center text-center">
                <h1 className="mt-4 text-4xl font-extrabold tracking-tight text-gray-900 sm:text-5xl">404</h1>
                <h2 className="mt-2 text-lg font-medium text-gray-600 sm:text-xl">Page not found</h2>
                <p className="mt-4 text-base text-gray-500">
                    Sorry, we couldn't find the page you're looking for.
                </p>
                <div className="mt-10">
                    <Link
                        to="/"
                        className="inline-flex items-center gap-2 rounded-md bg-gray-600 px-4 py-2.5 text-sm font-medium text-white transition-colors hover:bg-gray-800 focus:outline-none focus:ring-2 focus:ring-blue-500 focus:ring-offset-2"
                    >
                        <ArrowLeft className="h-4 w-4" />
                        Back to the Nest
                    </Link>
                </div>
            </div>
        </div>
    )
}
export default NoMatchPage
