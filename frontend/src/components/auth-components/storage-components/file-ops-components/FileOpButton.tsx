import React, { useEffect, useRef, useState, type JSX } from "react";
import type { SetBlurFunc } from "../../../ui/BackgroundBlur";

type FileOpButtonProps = {
    setBlur: SetBlurFunc
    setFileOp: (t: FileOperation) => void
}

type MenuButton = {
    text: string
    onClickFunc: (...args: any) => void,
}

export type FileOperation = null | "addFolder"

/**
 * Used to open the context menu to perform different file operations.
 * @returns The button for file operations
 */
export default function FileOpButton({setBlur, setFileOp}: FileOpButtonProps): JSX.Element{
    const [revealMenu, setRevealMenu] = useState(false);
    const menuDivRef = useRef(null);

    const MENU_BUTTONS_REF = useRef<Array<MenuButton>>([
        {
            text: "New folder",
            onClickFunc: () => {setFileOp("addFolder")},
        },
    ]);

    useMenuListener(setRevealMenu, menuDivRef);

    /**
     * A wrapper function used to hide the menu after a MenuButton.onClickFunc is
     * triggered.
     * 
     * All menu button functions must be wrapped in this.
     * 
     * @param f 
     */
    const hideMenuWrapper = (f: MenuButton["onClickFunc"], ...args: any) => {
        try{
            f(...args);
        }finally{
            setRevealMenu(false);
            setBlur(true);
        }
    }

    return (
        <div className="flex w-20 h-15 border">
            { revealMenu &&
                <div className="w-50 absolute p-2 bg-white border"
                ref={menuDivRef}>
                    <div>
                    {MENU_BUTTONS_REF.current.map((btn) => (
                        <button
                        className="w-full hover:bg-gray-400/50"
                        onClick={() => hideMenuWrapper(btn.onClickFunc)}>
                            {btn.text}
                        </button>
                    ))} 
                    </div>
                </div>
            }
            <button
            className="w-[inherit] h-[inherit]"
            onClick={() => setRevealMenu(true)}>
                New
            </button>
        </div>
    )
}

/**
 * A hook used to listen to any mouse clicks to unreveal the menu.
 */
function useMenuListener(menuSetter: (v: boolean) => void, menuRef: React.RefObject<HTMLDivElement|null>){
    useEffect(() => {
        const mouseDownListener = (e: MouseEvent) => {
            if(!menuRef.current || menuRef.current.contains(e.target as Node)){
                return;
            }
            menuSetter(false);
        }  

        document.addEventListener("mousedown", mouseDownListener);

        return () => {
            document.removeEventListener("mousedown", mouseDownListener);
        }
    }, [])
}
