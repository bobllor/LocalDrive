import { useEffect, useState, type JSX } from "react";
import { useLoaderData, useNavigate, useParams } from "react-router";
import { type User } from "../../../middleware";
import { fetchApi } from "../../../functions/fetchtils";
import { useFileStore, type File } from "../../../context/FileStore";
import FileListDisplay from "./FileListDisplay";

export default function StorageHome(): JSX.Element{
    const loaderData = useLoaderData<User>();
    const {setFiles, getFiles} = useFileStore();
    const navigate = useNavigate();
    const [files, setFilesState] = useState<Array<File>>([]);

    // :folderId param, will be either empty or with the route folder/:folderId
    let params = useParams();

    useEffect(() => {
        setFiles(params.folderId).then(() => {
            setFilesState(getFiles(params.folderId));
        }).catch((e) => {
            // TODO: log proper
            console.error(e, "an error occurred");
        })
    }, [params.folderId]);

    /**
     * Logouts the current validated user. This uses the session ID found
     * in the cookies.
     * 
     * If successful, it will logout the current user, invalidate the session in the
     * cookie, and redirect back to the home page.
     */
    async function logout(): Promise<void>{
        try{
            const res = await fetchApi<boolean>("/api/logout", "POST");

            if(res.output){
                navigate("/");
            }
        }catch(err){
            // TODO: add error popup here
            console.error(err);
        }
    }

    return (
        <>
            <div className="flex flex-col justify-center items-center">
                TEMPORARY: Hello {loaderData.username}. You are logged in.
                <button onClick={logout} className="border w-fit h-fit py-2 px-4">Logout</button>
                <FileListDisplay files={files} />
            </div> 
        </>
    )
}