import type { FileResponse } from "../../../../context/FileStore";
import { fetchApi } from "../../../../functions/fetchtils";

const KILOBYTE = 1024;

// obtained from request_types.go
export type FileUploadRequest = {
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
    // by default it will use 6 mb
    bytes: number = KILOBYTE * KILOBYTE * 6;

    /**
     * Begins the upload process. If successful, it will return a FileResponse
     * to be used to update the front end.
     * Any errors that occur will be thrown.
     * 
     * @param parentId The folder ID of the file upload, optional
     * @param uploadSessionId The upload session ID of the uploaded file
     * @param parentId The parent ID of the file, optional
     * @returns The FileResponse of the uploaded file if successful, otherwise undefined
     */
    async upload(file: File, uploadSessionId: string, parentId?: string): Promise<FileResponse | undefined>{
        if(!parentId){
            parentId = "";
        }
        let fileRes: FileResponse | undefined = undefined;

        try{
            const status = await this.uploadChunks(file, uploadSessionId);
            console.debug("Upload chunk status:", status);
            if(status){
                const uploadFile = await this.completeUpload(uploadSessionId);
                
                console.log("Upload file:", uploadFile);
                fileRes = uploadFile;
            }
        }catch(err){
            console.log("Uncaught error during file upload:", err);
            throw err;
        }

        return fileRes;
    }

    /**
     * Generates an upload ID. If the request is successful, it will
     * return the upload session ID. Otherwise, it will throw an error.
     * @param uploadReq The file upload request body
     * @returns The upload ID string
     */
    async generateUploadId(uploadReq: FileUploadRequest): Promise<string>{
        const res = await fetchApi<string>("/api/upload", "POST", uploadReq)
        
        if(res.status != "success"){
            throw res.error;   
        }

        console.debug("Generated ID response:", res);

        return res.output;
    }

    /**
     * Calls the API to fail the file entry in the back end based on the session
     * ID.
     * @param uploadSessionId The upload session ID
     */
    async failUpload(uploadSessionId: string){
        const res = await fetchApi<boolean>(`/api/upload/${uploadSessionId}/fail`, "PATCH");

        console.debug("Upload fail status:", res);
    }

    /**
     * Creates a new upload request body for use with fetch.
     * @param file The File that is being uploaded
     * @param parentId The parent ID of the file, optional
     * @returns 
     */
    newRequestBody(file: File, parentId?: string): FileUploadRequest{
        const fileNameExtObj = this.getNameAndExtension(file.name);
        if(!parentId){
            parentId = "";
        }

        const totalChunks = this.getTotalChunks(file);
        console.debug(`Total chunks for file ${file.name}:`, totalChunks);

        const reqBody: FileUploadRequest = {
            fileName: fileNameExtObj.name,
            fileSize: file.size,
            fileExtension: fileNameExtObj.extension,
            fileParentId: parentId,
            totalChunks: totalChunks,
        };

        return reqBody;
    }

    /**
     * Uploads the file in chunks.
     * @param file 
     * @param uploadId 
     * @returns The status of the upload
     */
    private async uploadChunks(file: File, uploadId: string): Promise<boolean>{
        let start = 0;
        let end = this.bytes;
        let chunkIndex = 0;
        
        const headers = {
            "Content-Type": "application/octet-stream",
        };

        while(start < file.size){
            if(end > file.size){
                // ensures that we dont go out of bounds
                end = file.size;
            }
            const apiEndpoint = `/api/upload/${uploadId}/${chunkIndex}`;
            const chunkBlob = file.slice(start, end);
            console.log(chunkBlob, chunkIndex + 1);

            const res = await fetchApi<any>(apiEndpoint, "POST", chunkBlob, headers);

            console.log(res);

            start += this.bytes;
            end += this.bytes;
            chunkIndex += 1;
        }

        return true;
    }

    /**
     * Finalizes the upload session.
     * @param uploadSessionId The upload ID used for the session
     * @returns The FileResponse of the uploaded file if successful, otherwise undefined
     */
    private async completeUpload(uploadSessionId: string): Promise<FileResponse | undefined>{
        const apiEndpoint = `/api/upload/${uploadSessionId}/complete`
        const res = await fetchApi<FileResponse>(apiEndpoint, "POST");

        if(res.status == "error"){
            console.error(`Failed to complete upload:`, res);
            return undefined;
        }

        return res.output;
    }

    /**
     * Extracts the file name and the extension separately as an object.
     * @param fileName The full file name with the extension
     * @returns 
     */
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