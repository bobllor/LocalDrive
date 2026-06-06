import { type JSX } from "react";
import type { SetBlurFunc } from "../../../ui/BackgroundBlur";
import ModalBase from "../../../ui/ModalBase";
import type React from "react";
import { useParams } from "react-router";
import { useFileStore } from "../../../../context/FileStore";

type AddFolderOpProps = {
    onClose: SetBlurFunc
}

const INPUT_NAME = "folder-input-field"

export default function AddFolderOp({onClose}: AddFolderOpProps): JSX.Element{
    const params = useParams();
    const addFolder = useFileStore(st => st.addFolder);

    const onFormSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
        e.preventDefault();

        try{
            const data = new FormData(e.currentTarget);
            const folderValue = data.get(INPUT_NAME) as string;

            const status = await addFolder(folderValue, params.folderId);

            console.log(status);
        }finally{
            onClose(false);
        }
    }

    return (
        <ModalBase>
            <div>
                <form
                onSubmit={onFormSubmit}>
                    <input type="text" className="border" name={INPUT_NAME} autoComplete="off" />
                </form>
            </div>
        </ModalBase>        
    )
}