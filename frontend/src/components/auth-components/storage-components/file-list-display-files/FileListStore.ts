import { create } from "zustand";

type FileListStore = {
    selectedFileIds: Set<string>,
    /**
     * Adds a file ID to the set.
     * @param id 
     */
    addFileId: (id: string) => void,
    /**
     * Clears all contents of the selected files set.
     */
    clearFileIds: () => void,
}

export const useFileListStore = create<FileListStore>((set, get) => ({
    selectedFileIds: new Set<string>(),
    addFileId: (id: string) => {
        if(!get().selectedFileIds.has(id)){
            set(state => {
                const baseSet = new Set<string>(state.selectedFileIds);
                baseSet.add(id);

                return {
                    ...state, 
                    selectedFileIds: baseSet,
                }
            })
        }
    },
    clearFileIds: () => {
        set(state => ({...state, selectedFileIds: new Set<string>()}));
    },
}));