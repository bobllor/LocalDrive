import type { JSX } from "react";
import type { SetBlurFunc } from "../../ui/BackgroundBlur";
import type React from "react";
import { fetchApi } from "../../../functions/fetchtils";

type RenameFileProps = {
    onClose: SetBlurFunc
    /**
     * The ID of the file. This is a lifted state from the FileListDisplay rows.
     */
    fileId: string
}

type RequestRenameFile = {
    fileId: string
    newFileName: string
}

const FILE_ELEMENT_NAME = "file-name";

export default function RenameFile({onClose, fileId}: RenameFileProps): JSX.Element{
    const onSubmitRenameFile = async (event: React.SubmitEvent<HTMLFormElement>) => {
        event.preventDefault();

        try{
            const data = new FormData(event.currentTarget);
            const fileName = data.get(FILE_ELEMENT_NAME) as string;

            const reqObj: RequestRenameFile = {
                fileId: fileId,
                newFileName: fileName,
            };

            const res = await fetchApi("/api/file/rename", "PATCH", reqObj);

            console.log(res);
        }finally{
            onClose(false);
        }
    }

    return (
        <form
        className="flex flex-col justify-center items-center"
        onSubmit={onSubmitRenameFile}>
            <input 
            autoComplete="off"
            type="text" className="border" name={FILE_ELEMENT_NAME} id={FILE_ELEMENT_NAME} />
            <div
            className="flex items-center justify-center gap-3">
                <button
                type="button"
                onClick={() => onClose(true)}>
                    Cancel
                </button>
                <button 
                type="submit">
                    Submit
                </button>
            </div>
        </form>
    )
}