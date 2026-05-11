import { create } from "zustand";
import { fetchApi } from "../functions/fetchtils";

type FileStore = {
    files: Record<string, Array<File>>,
    /**
     * Sets the contents of the files based on the parentID. If the parentID already
     * has an entry, then this will do nothing.
     * 
     * An error can occur and will return a ResponseApi error.
     * @param parentID 
     */
    setFiles: (parentID?: string) => Promise<void>,
    /**
     * Retrieves the files based on the parentID.
     * @param parentID The parentID of the files, this can be null indicating it is the root folder
     * @returns The array of the files related to the parentID
     */
    getFiles: (parentID?: string) => Array<File>,
}

/**
 * The type representing the File data of the database.
 * This does not include the file path.
 */
export type File = {
    accountID: string
    fileName: string
    fileType: string
    fileID: string
    parentID: string
    fileSize: number
    modifiedOn: Date
    deletedOn?: Date
}

const ROOT_KEY = "root";

export const useFileStore = create<FileStore>((set, get) => ({
    files: {},
    setFiles: async (parentID?: string) => {
        const key = parentID ? parentID : ROOT_KEY;
        const route = parentID ? `/api/storage/folder/${key}` : "/api/storage";
        const baseFiles = get().files;

        // will not update the state if it already exists
        if(key in baseFiles){
            // TODO: REMOVE IN PROD
            console.log("Key already exists, skipping content");
            return;
        }

        try{
            const newFiles = await fetchApi<Array<File>>(route);

            const newObj: Record<string, File[]> = {};
            newObj[key] = newFiles.output;

            set(state => ({state, files: {...state.files, ...newObj}}));
            // TODO: remove this or something idk
            console.debug(`New file store size: ${Object.keys(get().files).length}`);
        }catch(e){
            throw e;
        }
    },
    updateFiles: () => {},
    getFiles: (parentID?: string) => {
        // TODO: log properly
        const files = get().files;
        const key = parentID ? parentID : ROOT_KEY;
        console.debug(`Parent ID: ${key}`);

        return files[key];
    }
}));
