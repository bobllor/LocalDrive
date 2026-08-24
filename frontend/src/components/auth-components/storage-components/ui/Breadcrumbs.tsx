import { useEffect, useState, type JSX } from "react";
import { useParams } from "react-router";
import { fetchApi } from "../../../../functions/fetchtils";
import { FaAngleRight } from "react-icons/fa6";

type BreadcrumbFile = {
    fileId: string
    parentId: string
    fileName: string
}

type BreadcrumbObject = {
    folderName: string
    fileId: string
    url: string
}

const DEFAULT_HOME: BreadcrumbObject = {
    folderName: "Home",
    fileId: "",
    url: "/storage",
};

export default function Breadcrumbs(): JSX.Element{
    const breadcrumbs = useBreadcrumbs();
    const params = useParams();
    
    return (
        <div className="flex items-center justify-center">
            {breadcrumbs.map((obj, i) => (
                <nav className="flex items-center justify-center"
                key={obj.fileId}>
                    <div 
                    className="p-2 flex justify-center items-center hover:bg-gray-400/50 select-none"
                    key={i}>
                        {obj.fileId == params.folderId || !params.folderId
                        ? <span>{obj.folderName}</span>
                        : <a href={obj.url}>{obj.folderName}</a>
                        }
                    </div>
                    {i != breadcrumbs.length - 1 &&
                        <div className="">
                            <FaAngleRight size={15}/>
                        </div>
                    }
                </nav>
            ))}
        </div>
    )
}

function useBreadcrumbs(): Array<BreadcrumbObject>{
    const params = useParams();

    const [breadcrumbs, setBreadcrumbs] = useState<Array<BreadcrumbObject>>([]);

    useEffect(() => {
        const getBreadcrumbs = async () => {
            // if undefined, we are in root. root will reset access back to the home.
            if(!params.folderId){
                setBreadcrumbs([DEFAULT_HOME]);
                return;
            }

            const res = await fetchApi<Array<BreadcrumbFile>>(`/api/folders/${params.folderId}/breadcrumbs`, "GET");
            // ensures we rebuild from the root
            // will always consist of a minimum one: home
            const arr: Array<BreadcrumbObject> = [DEFAULT_HOME];
            res.output.forEach(obj => {
                const bcObj: BreadcrumbObject = {
                    folderName: obj.fileName,
                    fileId: obj.fileId,
                    url: `/storage/folder/${obj.fileId}`,
                };

                arr.push(bcObj);

                setBreadcrumbs(arr);
            });
        }

        getBreadcrumbs();
    }, [params]);

    return breadcrumbs;
}