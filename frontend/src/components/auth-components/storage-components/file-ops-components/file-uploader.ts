import { fetchApi } from "../../../../functions/fetchtils";
import type { ResponseApi } from "../../../../response-types";

const KILOBYTE = 1024;

// obtained from request_types.go
type FileUploadRequest = {
    fileName: string
    fileSize: number
    fileExtension: string
    fileParentId: string
    totalChunks: number
}

type FileNameExtension = {
    name: string
    extension: string
}

/**
 * Class used to handle file uploading.
 */
export class FileUploader{
    files: FileList;
    // by default it will use 6 mb
    bytes: number = KILOBYTE * KILOBYTE * 6;

    constructor(file: FileList){
        this.files = file;
    }

    async upload(parentId?: string){
        if(!parentId){
            parentId = "";
        }

        for(let i = 0; i < this.files.length; i++){
            const file = this.files.item(i);
            if(!file){
                console.error(`File (index ${i}) is null from FileList:`, this.files);
                continue;
            }

            const totalChunks = this.getTotalChunks(file);
            console.debug(`Total chunks for file ${file.name}:`, totalChunks);

            const fileNameExtObj = this.getNameAndExtension(file.name);

            const reqBody: FileUploadRequest = {
                fileName: fileNameExtObj.name,
                fileSize: file.size,
                fileExtension: fileNameExtObj.extension,
                fileParentId: parentId,
                totalChunks: totalChunks,
            };

            console.log(reqBody);
        }
    }

    /**
     * Returns the length of the FileList.
     * @returns The files length
     */
    length(): number{
        return this.files.length;
    }

    /**
     * Generates an upload ID.
     */
    private async generateUploadId(uploadReq: FileUploadRequest): Promise<string>{
        const res = await fetchApi<string>("/api/upload", "POST", uploadReq)

        return res.output;
    }

    private getNameAndExtension(fileName: string): FileNameExtension{
        const obj: FileNameExtension = {
            name: "",
            extension: "",
        };

        const nameArr = fileName.split(".");
        if(nameArr.length == 0){
            return obj;
        }else if(nameArr.length == 1){
            obj.name = nameArr.at(0)!;
            obj.extension = ""; 
        }else{
            obj.name = nameArr.slice(0, -1).join(".");
            // does not matter, the back end will strip
            // the leading period regardless
            obj.extension = nameArr.at(-1)!;
        }

        return obj
    }

    /**
     * Retrieves the total chunks expected for the uploaded file
     * @param file The File from the FileList
     * @returns The total chunks required to upload
     */
    private getTotalChunks(file: File): number{
        let chunks = 1;
        const fileSize = file.size;

        if(fileSize > 0){
            chunks = Math.ceil(file.size / this.bytes);
        }

        return chunks;
    }
}