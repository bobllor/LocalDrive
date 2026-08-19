import type { JSX } from "react";

type NavigationObject = {
    displayName: string
    name: string
    url: string
};

const navs: Array<NavigationObject> = [
    {
        displayName: "Drive",
        name: "drive",
        url: "/storage",
    },
    {
        displayName: "Trash",
        name: "trash",
        url: "/storage/trash",
    },
];

export function NavigationMenu(): JSX.Element{
    return (
        <div className="w-30 p-2">
            {
                navs.map((obj) => 
                    <div key={obj.name}>
                        <button>
                            <a href={obj.url}>
                                {obj.displayName}
                            </a>
                        </button>
                    </div>
                )
            }
        </div>
    )
}