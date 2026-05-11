import type { JSX } from "react";
import type { File } from "../../../context/FileStore";
import { useNavigate } from "react-router";
import React from "react";
import { useFileListStore } from "./file-list-display-files/FileListStore";
import { useShallow } from "zustand/shallow";

const THEAD_ELEMENTS: Array<TableHeadObj> = [
    {
        text: "Name",
    },
    {
        text: "File size",
    },
];

const hoverCss = "hover:bg-gray-400/45";

export default function FileListDisplay({files}: FileListDisplayProps): JSX.Element{
    const clearFileIds = useFileListStore(state => state.clearFileIds);

    return (
        <table 
        className="table-auto w-full">
            <thead>
                <tr className="select-none">
                    {THEAD_ELEMENTS.map((tObj, i) => 
                        <th 
                        onClick={clearFileIds}
                        className={hoverCss}
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
                    files.map((fileObj, i) => 
                        <React.Fragment key={i}>
                            <FileTableRow fileObj={fileObj} />
                        </React.Fragment>
                    )
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
                navigate(`folder/${fileObj.fileID}`); 
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
        </>
    )
}

type TableHeadObj = {
    text: string,
}

type FileListDisplayProps = {
    files: Array<File>,
}

type FileObjProps = {
    fileObj: File,
}