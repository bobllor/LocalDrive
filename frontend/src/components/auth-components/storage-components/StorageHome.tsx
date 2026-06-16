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
                <input type="file" hidden onChange={
                    () => onInputFileChangeUploadFile(inputUploadFileRef, params.folderId)
                } 
                    ref={inputUploadFileRef} />
            </div> 
        </>
    )
}

/**
 * Handles uploading a file to the backend.
 * @param ref The reference object of the file uploading input element
 * @param parentId The parent ID of the file, can be undefined or empty
 * @returns 
 */
async function onInputFileChangeUploadFile(ref: RefObject<HTMLInputElement | null>, parentId?: string){
    if(!ref.current){
        return;
    }
    const inputEle = ref.current;
    if(!ref.current.files){
        return;
    }

    const addFileResponse = useFileStore.getState().addFileResponse;
    const uploader = new FileUploader();

    try{
        const files = ref.current.files;

        for(let i = 0; i < files.length; i++){
            const file = files.item(i);
            if(!file){
                console.error(`File (index ${i}) is null from FileList:`, files);
                continue;
            }

            const reqBody = uploader.newRequestBody(file, parentId);
            
            const uploadSessionId = await uploader.generateUploadId(reqBody);

            try{
                const fileRes = await uploader.upload(file, uploadSessionId, parentId);
                if(fileRes !== undefined){
                    addFileResponse(fileRes, parentId);
                }
            }catch(uploadErr){
                // TODO: set the upload state to fail on the UI
                // errors are handled in the uploader.
                if(uploadSessionId != undefined){
                    await uploader.failUpload(uploadSessionId);
                }
                console.error("An error occurred while uploading:", uploadErr);
            }
        }
    }catch(err){
        // TODO: add an upload UI/UX loader thingy, or in other words
        // the component that displays the file progress. also includes retries
        // and failures.
        // TODO: the chunk index, where it failed, and the File object needs to be tracked.
        console.error("An uncaught error occurred during uploading:", err);
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