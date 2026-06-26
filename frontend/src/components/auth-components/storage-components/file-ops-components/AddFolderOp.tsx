import { useEffect, useRef, type JSX } from "react";
import type { SetBlurFunc } from "../../../ui/BackgroundBlur";
import type React from "react";
import { useParams } from "react-router";
import { useFileStore } from "../../../../context/FileStore";

type AddFolderOpProps = {
    onClose: SetBlurFunc
}

const INPUT_NAME = "folder-input-field";
const BUTTON_CLASS = "hover:bg-blue-300/60 rounded-2xl p-2 transition";

export default function AddFolderOp({onClose}: AddFolderOpProps): JSX.Element{
    const params = useParams();
    const addFolder = useFileStore(st => st.addFolder);

    const inputRef = useRef<HTMLInputElement|null>(null);
    useAutoSelectInput(inputRef);

    const onFormSubmit = async (e: React.SubmitEvent<HTMLFormElement>) => {
        e.preventDefault();

        try{
            const data = new FormData(e.currentTarget);
            const folderValue = data.get(INPUT_NAME) as string;

            // due to the await, this is used just in case.
            onClose(false);
            const status = await addFolder(folderValue, params.folderId);

            console.log(status);
        }finally{
            onClose(false);
        }
    }

    return (
        <form
        className="flex flex-col justify-between items-center gap-3"
        onSubmit={onFormSubmit}>
            <input 
            ref={inputRef}
            type="text" className="border" name={INPUT_NAME} autoComplete="off" defaultValue={"New folder"} />
            <div className="flex justify-center items-center gap-6">
                <button 
                className={BUTTON_CLASS}
                type="button" onClick={() => onClose(false)}>Cancel</button>
                <button 
                className={BUTTON_CLASS}
                type="submit">Create</button>
            </div>
        </form>
    )
}

/**
 * A hook used to auto focus and select the input text for typing on load
 * @param ref The RefObject reprenting the input element
 */
function useAutoSelectInput(ref: React.RefObject<HTMLInputElement | null>){
    useEffect(() => {
        if(ref.current){
            ref.current.focus(); 
            ref.current.select();
        }
    }, []);
}