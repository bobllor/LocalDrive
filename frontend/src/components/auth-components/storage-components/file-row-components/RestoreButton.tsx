import type { JSX } from "react";
import { FILE_ROW_BUTTON_CSS } from "./vars";
import { fileStoreKeys, useFileStore, type FileResponse } from "../../../../context/FileStore";
import { fetchApi } from "../../../../functions/fetchtils";

type RestoreButtonProps = {
    fileObj: FileResponse
}

export default function RestoreButton({fileObj}: RestoreButtonProps): JSX.Element{
    return (
        <button
        className={FILE_ROW_BUTTON_CSS}
        onClick={() => restoreFile(fileObj)}>
            RES
        </button>
    )
}

/**
 * Restores a deleted file to its parent folder. If the parent folder
 * does not exist, then the file will be updated to the root folder.
 * 
 * @param fileId The file ID of the file to be restored
 */
async function restoreFile(fileObj: FileResponse): Promise<void>{
    const removeFile = useFileStore.getState().removeFile;
    // NOTE: when a file is deleted, it gets removed from the original
    // array of the parent. this will either restore it back to the
    // parent or to root if the parent is unavailable
    const addFile = useFileStore.getState().addFileResponse;
    
    const route = `/api/file/restore/${fileObj.fileID}`;

    // is successful, it will always contain a minimum of 1 output
    // due to this being a single file restoration, it will always be 1.
    const res = await fetchApi<Array<FileResponse>>(route, "PATCH");
    if(res.status != "success"){
        // TODO: add error toast here!
        console.error("Failed to restore files", res);
        return;
    }

    const file = res.output[0];

    removeFile(fileObj.fileID, fileStoreKeys.trash);
    await addFile(file, file.parentID);
}