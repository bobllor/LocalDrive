import { useEffect, useState, type JSX } from "react";
import { useNavigate, useParams } from "react-router";
import { fetchApi } from "../../../functions/fetchtils";
import { useFileStore, type FileResponse } from "../../../context/FileStore";
import FileListDisplay from "./FileListDisplay";
import FileOpButton from "./file-ops-components/FileOpButton";

export default function StorageHome(): JSX.Element{
    const navigate = useNavigate();

    const files = useFileDisplay();

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
            <div className="flex flex-col justify-center items-center gap-1">
                <FileOpButton />
                <button onClick={logout} className="border w-fit h-fit py-2 px-4">Logout</button>
                <div className="border w-full">
                    <FileListDisplay files={files} />
                </div>
            </div> 
        </>
    )
}

function useFileDisplay(): Array<FileResponse>{
    // :folderId param, will be either empty or with the route folder/:folderId
    let params = useParams();

    const {setFiles, getFiles} = useFileStore();
    const [files, setFilesState] = useState<Array<FileResponse>>([]);

    useEffect(() => {
        setFiles(params.folderId).then(() => {
            setFilesState(getFiles(params.folderId));
        }).catch((e) => {
            // TODO: log proper
            // will need to redirect this
            console.error(e, "an error occurred");
        })
    }, [params.folderId]);

    return files;
}
