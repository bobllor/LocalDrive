import type { JSX } from "react";
import type { FileResponse } from "../../../context/FileStore";
import { useNavigate } from "react-router";
import React from "react";
import { useFileListStore } from "./store/FileListStore";
import { useShallow } from "zustand/shallow";
import type { SetBlurFunc } from "../../ui/BackgroundBlur";
import type { ModalOperation } from "./StorageHome";
import DownloadButton from "./file-row-components/DownloadButton";
import RenameButton from "./file-row-components/RenameButton";

type TableHeadObj = {
    text: string
}

type FileListDisplayProps = {
    files?: Array<FileResponse>
    setBlur: SetBlurFunc
    setModalOp: (op: ModalOperation) => void
    setFileId: (s: string) => void
}

type FileObjProps = {
    fileObj: FileResponse
    setBlur: SetBlurFunc
    setModalOp: (op: ModalOperation) => void
    setFileId: (s: string) => void
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

export default function FileListDisplay({files, setBlur, setModalOp, setFileId}: FileListDisplayProps): JSX.Element{
    const clearFileIds = useFileListStore(state => state.clearFileIds);

    // files in progress will not be updated
    const filesMap = files ? files.filter(fileObj => fileObj.uploadStatus === "completed") : [];

    return (
        <table 
        className="table-auto w-full">
            <thead>
                <tr className="select-none">
                    {THEAD_ELEMENTS.map((tObj, i) => 
                        <th 
                        onClick={clearFileIds}
                        className=""
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
                            <FileTableRow fileObj={fileObj} setModalOp={setModalOp} setBlur={setBlur} setFileId={setFileId} />
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
function FileTableRow({fileObj, setBlur, setModalOp, setFileId}: FileObjProps): JSX.Element{
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
            <FileTableData fileObj={fileObj} setBlur={setBlur} setModalOp={setModalOp} setFileId={setFileId} />
        </tr>
    )
}

/**
 * Component that represents any non-directory file table data.
 */
function FileTableData({fileObj, setBlur, setModalOp, setFileId}: FileObjProps): JSX.Element{
    const cssClass = "";

    const renameOnClick = (fileId: string) => {
        setModalOp("renameFile");
        setBlur(true);
        setFileId(fileId);
    }

    return (
        <>
            <td className={cssClass}>
                {fileObj.fileName}
            </td>
            <td className={cssClass}>
                {fileObj.fileType != "dir" ? fileObj.fileSize : "-"}
            </td>
            <td className={"flex"}>
                {
                    fileObj.fileType != "dir" && <DownloadButton fileObj={fileObj} />
                }
                <RenameButton fileObj={fileObj} renameFunction={renameOnClick} />
            </td>
        </>
    )
}