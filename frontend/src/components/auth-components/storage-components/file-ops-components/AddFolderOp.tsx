import type { JSX } from "react";
import type { FileResponse } from "../../../../context/FileStore";
import { fetchApi } from "../../../../functions/fetchtils";
import type { ResponseApi } from "../../../../response-types";
import type { SetBlurFunc } from "../../../ui/BackgroundBlur";

type AddFolderOpProps = {
    setBlur: SetBlurFunc
}

export default function AddFolderOp({setBlur}: AddFolderOpProps): JSX.Element{
    return (
        <div className="bg-white w-10 h-10">

        </div>
    )
}

/**
 * Adds a folder. 
 * 
 * The response will return a FileResponse object.
 */
async function addFolder(fileName: string, parentId?: string){
    const reqBody = {
        fileName: fileName,
        parentId: parentId,
    }

    const res: ResponseApi<FileResponse> = await fetchApi("/api/folders/add", "POST", reqBody);
}