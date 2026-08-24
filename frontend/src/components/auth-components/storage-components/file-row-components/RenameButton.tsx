import type { JSX } from "react";
import { FILE_ROW_BUTTON_CSS } from "./vars";
import type { FileResponse } from "../../../../context/FileStore";

export default function RenameButton({fileObj, renameFunction}: RenameButtonProps): JSX.Element{
    return (
        <button
        onClick={() => renameFunction(fileObj.fileID)}
        className={FILE_ROW_BUTTON_CSS}>
            R
        </button>
    )
}

type RenameButtonProps = {
    fileObj: FileResponse
    /**
     * The function that triggers the renaming process.
     * @param fileId The file ID that is being renamed
     * @returns 
     */
    renameFunction: (fileId: string) => void
}