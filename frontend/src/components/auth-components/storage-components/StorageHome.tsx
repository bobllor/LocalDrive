import { useEffect, useState, type JSX } from "react";
import { useNavigate, useParams } from "react-router";
import { fetchApi } from "../../../functions/fetchtils";
import { useFileStore, type FileResponse } from "../../../context/FileStore";
import FileListDisplay from "./FileListDisplay";
import FileOpButton, { type FileOperation } from "./file-ops-components/FileOpButton";
import BackgroundBlur from "../../ui/BackgroundBlur";
import AddFolderOp from "./file-ops-components/AddFolderOp";
import ModalBase from "../../ui/ModalBase";

export default function StorageHome(): JSX.Element{
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

    const navigate = useNavigate();
    const files = useFileDisplay();

    const [showBlur, setShowBlur] = useState(false);
    const [fileOp, setFileOp] = useState<FileOperation>(null);

    /**
     * Closes the background blur.
     */
    const onClose = () => setShowBlur(false);

    return (
        <>
            {showBlur &&
                <BackgroundBlur setBlur={setShowBlur}>
                    <ModalBase>
                        {fileOp == "addFolder" && <AddFolderOp onClose={onClose} />}
                    </ModalBase>
                </BackgroundBlur>
            }
            <div className="flex flex-col justify-center items-center gap-1">
                <FileOpButton setBlur={setShowBlur} setFileOp={setFileOp} />
                <button onClick={logout} className="border w-fit h-fit py-2 px-4">Logout</button>
                <div className="border w-full">
                    {files !== undefined
                    ? <FileListDisplay files={files} />
                    : <div>Loading...</div>
                    }
                </div>
            </div> 
        </>
    )
}

/**
 * A hook that retrieves the files from the API with the given folder ID
 * from the URL parameters.
 * 
 * @returns An array of FileResponses
 */
function useFileDisplay(): Array<FileResponse>{
    // :folderId param, will be either empty or with the route folder/:folderId
    let params = useParams();

    const {setFiles, getFiles} = useFileStore();
    const files = getFiles(params.folderId);

    useEffect(() => {
        setFiles(params.folderId);
    }, [params.folderId]);

    return files;
}