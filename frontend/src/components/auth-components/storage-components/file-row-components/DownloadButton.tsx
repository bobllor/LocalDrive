import type { JSX } from "react";
import { createUrl } from "../../../../server-utils";
import type { FileResponse } from "../../../../context/FileStore";
import { FILE_ROW_BUTTON_CSS } from "./vars";

export default function DownloadButton({fileObj}: {fileObj: FileResponse}): JSX.Element{
    return (
        <span
        className={FILE_ROW_BUTTON_CSS}
        onClick={() => downloadFile(fileObj.fileID)}>
            D
        </span>
    )
}

function downloadFile(fileId: string): void{
    let a = window.document.createElement("a");
    const fileUrl = createUrl(`/api/download/file/${fileId}`);

    // hidden file trick, i did not know this!
    a.href = fileUrl;
    a.style.display = "none";

    document.body.appendChild(a);

    a.click();
    document.body.removeChild(a);
}