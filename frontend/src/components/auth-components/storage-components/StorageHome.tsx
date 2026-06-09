import { useEffect, useRef, useState, type JSX, type RefObject } from "react";
import { useNavigate, useParams } from "react-router";
import { fetchApi } from "../../../functions/fetchtils";
import { useFileStore, type FileResponse } from "../../../context/FileStore";
import FileListDisplay from "./FileListDisplay";
import FileOpButton, { type FileOperation } from "./file-ops-components/FileOpButton";
import BackgroundBlur from "../../ui/BackgroundBlur";
import AddFolderOp from "./file-ops-components/AddFolderOp";
import ModalBase from "../../ui/ModalBase";
import { FileUploader } from "./file-ops-components/file-uploader";

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
    const [fileOp, setFileOp] = useState<FileOperation>("");

    const inputUploadFileRef = useRef<HTMLInputElement | null>(null);
    const params = useParams();

    /**
     * Closes the background blur.
     */
    const onClose = () => setShowBlur(false);

    return (
        <>
            {showBlur && fileOp != "" &&
                <BackgroundBlur setBlur={setShowBlur}>
                    <ModalBase>
                        {fileOp == "addFolder" && <AddFolderOp onClose={onClose} />}
                    </ModalBase>
                </BackgroundBlur>
            }
            <div className="flex flex-col justify-center items-center gap-1">
                <FileOpButton setBlur={setShowBlur} setFileOp={setFileOp} fileUploadRef={inputUploadFileRef} />
                <button onClick={logout} className="border w-fit h-fit py-2 px-4">Logout</button>
                <div className="border w-full">
                    {files !== undefined
                    ? <FileListDisplay files={files} />
                    : <div>Loading...</div>
                    }
                </div>
                {/* file is hidden, the FileOpButton will trigger the upload */}
                <input type="file" hidden onChange={() => onInputFileChange(inputUploadFileRef, params.folderId)} ref={inputUploadFileRef} />
            </div> 
        </>
    )
}

/**
 * Handles uploading a file to the backend. If 
 * @param ref The reference object of the file uploading input element
 * @param parentId The parent ID of the file, can be undefined or empty
 * @returns 
 */
async function onInputFileChange(ref: RefObject<HTMLInputElement | null>, parentId?: string){
    if(!ref.current){
        return;
    }
    const inputEle = ref.current;
    if(!ref.current.files){
        return;
    }

    try{
        const uploader = new FileUploader(ref.current.files);

        await uploader.upload(parentId);
    }finally{
        // resets the input file
        inputEle.value = "";
    }
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