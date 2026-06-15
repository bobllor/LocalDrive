import type { JSX } from "react";
import type { FileResponse } from "../../../context/FileStore";
import { useNavigate } from "react-router";
import React from "react";
import { useFileListStore } from "./store/FileListStore";
import { useShallow } from "zustand/shallow";
import { createUrl } from "../../../server-utils";

type TableHeadObj = {
    text: string,
}

type FileListDisplayProps = {
    files: Array<FileResponse>,
}

type FileObjProps = {
    fileObj: FileResponse,
}

const THEAD_ELEMENTS: Array<TableHeadObj> = [
    {
        text: "Name",
    },
    {
        text: "File size",
    },
    {
        text: "",
    },
];

const hoverCss = "hover:bg-gray-400/45";

export default function FileListDisplay({files}: FileListDisplayProps): JSX.Element{
    const clearFileIds = useFileListStore(state => state.clearFileIds);

    // files in progress will not be updated
    const filesMap = files.filter(fileObj => {
        return fileObj.uploadStatus === "completed";
    });

    return (
        <table 
        className="table-auto w-full">
            <thead>
                <tr className="select-none">
                    {THEAD_ELEMENTS.map((tObj, i) => 
                        <th 
                        onClick={clearFileIds}
                        className="w-full"
                        key={i}>
                            <span>
                                {tObj.text}
                            </span>
                        </th>
                    )}
                </tr>
            </thead>
            <tbody>
                {
                    filesMap.map(fileObj => (
                        <React.Fragment key={fileObj.fileID}>
                            <FileTableRow fileObj={fileObj} />
                        </React.Fragment>
                    ))
                }
            </tbody>
        </table>
    )
}

/**
 * Component that handles the table data for the object.
 */
function FileTableRow({fileObj}: FileObjProps): JSX.Element{
    const {selectedFileIds, addFileId, clearFileIds} = useFileListStore(useShallow(state => ({
        selectedFileIds: state.selectedFileIds,
        addFileId: state.addFileId,
        clearFileIds: state.clearFileIds,
    })));
    const navigate = useNavigate();

    return (
        <tr 
        onClick={(e) => {
            if(!e.ctrlKey){
                clearFileIds();
            }
            addFileId(fileObj.fileID);
        }}
        onDoubleClick={() => {
            if(fileObj.fileType == "dir"){
                navigate(`/storage/folder/${fileObj.fileID}`); 
            }
        }}
        className={`select-none ${selectedFileIds.has(fileObj.fileID) ? "bg-blue-400/60 hover:bg-blue-400/80" : hoverCss}`}>
            <FileTableData fileObj={fileObj} />
        </tr>
    )
}

/**
 * Component that represents any non-directory file table data.
 */
function FileTableData({fileObj}: FileObjProps): JSX.Element{
    const cssClass = "";

    return (
        <>
            <td className={cssClass}>
                {fileObj.fileName}
            </td>
            <td className={cssClass}>
                {fileObj.fileType != "dir" ? fileObj.fileSize : "-"}
            </td>
            <td className={cssClass}>
                {
                    fileObj.fileType != "dir" &&
                    <span
                    className="hover:bg-gray-500 rounded-2xl px-1.5 flex justify-center items-center"
                    onClick={() => downloadFile(fileObj.fileID)}>
                        D
                    </span>
                }
            </td>
        </>
    )
}

function downloadFile(fileId: string): void{
    let a = window.document.createElement("a");
    const fileUrl = createUrl(`/api/download/file/${fileId}`);

    a.href = fileUrl;
    a.style.display = "none";

    document.body.appendChild(a);

    a.click();
    document.body.removeChild(a);
}