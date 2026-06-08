import { create } from "zustand";
import { fetchApi } from "../functions/fetchtils";
import type { ResponseApi, ResponseStatus } from "../response-types";

type FileStore = {
    /**
     * A map of parent IDs and their files.
     */
    files: Record<string, Array<FileResponse>>
    /**
     * Sets the contents of the files based on the parent ID. If the parent ID already
     * has an entry, then this will do nothing.
     * 
     * An error can occur and will return a ResponseApi error.
     * @param parentId
     */
    setFiles: (parentId?: string) => Promise<void>
    /**
     * Retrieves the files based on the parent ID.
     * @param parentId The parent ID of the files, this can be null indicating it is the root folder
     * @returns The array of the files related to the parentID
     */
    getFiles: (parentId?: string) => Array<FileResponse>
    /**
     * Adds a new folder to the database. Upon a successful request, it will update
     * the file state of the parent ID.
     * 
     * It will return a success or error depending on the API response.
     * 
     * @param folderName The folder name
     * @param parentId The parent ID the folder resides in, it is optional
     * @returns 
     */
    addFolder: (folderName: string, parentId?: string) => Promise<ResponseStatus>,
}

/**
 * The type representing the File data of the database.
 * This does not include the file path or account owner.
 */
export type FileResponse = {
    fileName: string
    fileType: string
    fileID: string
    extension: string
    parentID: string
    fileSize: number
    modifiedOn: Date
    deletedOn?: Date
}

export const useFileStore = create<FileStore>((set, get) => ({
    files: {},
    setFiles: async (parentId ?: string) => {
        const key = getParentIdUndefined(parentId);
        const route = parentId ? `/api/storage/folder/${key}` : "/api/storage";
        const baseFiles = get().files;

        // will not update the state if it already exists
        if(key in baseFiles){
            // TODO: REMOVE IN PROD
            console.log("Key already exists, skipping content");
            return;
        }

        try{
            const newFiles = await fetchApi<Array<FileResponse>>(route);

            const newObj: Record<string, FileResponse[]> = {};
            newObj[key] = newFiles.output;

            set(state => ({...state, files: {...state.files, ...newObj}}));
            // TODO: remove this or something idk
            console.debug(`New file store size: ${Object.keys(get().files).length}`);
        }catch(e){
            throw e;
        }
    },
    getFiles: (parentId?: string) => {
        // TODO: log properly
        const files = get().files;
        const key = getParentIdUndefined(parentId);
        console.debug(`Parent ID: ${key}`);

        return files[key];
    },
    addFolder: async (folderName: string, parentId?: string) => {
        const reqBody = {
            fileName: folderName,
            parentId: parentId,
        }

        const res: ResponseApi<FileResponse> = await fetchApi<FileResponse>("/api/folders/add", "POST", reqBody);
        console.log("Add folder response:", res);

        if(res.status == "success"){
            const key = getParentIdUndefined(parentId);
            const files = get().getFiles(parentId).map(v => v);

            files.push(res.output);

            set(st => ({...st, files: {...st.files, [key]: files}}));
        }

        return res.status;
    },
}));

/**
 * Checks the parent ID and returns the parent ID if it is not undefined, otherwise
 * it will return an empty value.
 * 
 * @param parentId A string representing the parent ID, this can be undefined
 * @returns 
 */
function getParentIdUndefined(parentId?: string): string{
    // kept as a "just in case" helper
    // as of 6/7/2026 nil/undefined is no longer allowed for parent IDs,
    // an empty parent ID is considered to be the root
    return parentId ? parentId : "";
}