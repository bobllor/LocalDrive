import type { JSX } from "react";
import { useFileStore, type FileResponse } from "../../../../context/FileStore";
import { FILE_ROW_BUTTON_CSS } from "./vars";
import { fetchApi } from "../../../../functions/fetchtils";

export default function DeleteButton({fileObj}: {fileObj: FileResponse}): JSX.Element{
    return (
        <button
        className={FILE_ROW_BUTTON_CSS}
        onClick={() => deleteFile(fileObj.fileID, fileObj.parentID)}>
            DEL
        </button>
    )
}

/**
 * Marks a file to be deleted. Upon success, it will update the folder storing
 * the file in the context.
 * A toast will also appear on success.
 * 
 * @param fileId 
 * @returns A boolean indicating a success or failure in deletion
 */
async function deleteFile(fileId: string, parentId?: string): Promise<void>{
    const res = await fetchApi<boolean>(`/api/file/delete/${fileId}`, "DELETE");
    // TODO: redirect to an error page or toast, depending on the issue
    if(res.status === "error"){  
        console.error("TODO: error deleting file:", res);
        return;
    }
    
    if(res.output){
        // TODO: toast success, will need to have an "undo" button
        // in the toast as well.
        // react hot toast has a custom() method that can be used for this
        const file = useFileStore.getState().removeFile(fileId, parentId);
        console.debug("Removed file:", file);
        
        if(file !== undefined){
            // TODO: toast here
            console.log("file has been removed");
        }

    }else{
        // TODO: anything other than 200-300 will probably be a redirect.
    }
}