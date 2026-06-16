import React, { useEffect, useRef, type JSX, type RefObject } from "react";

/**
 * A callback function used to set the state of the background blur.
 * 
 * @param st The boolean state to set the render condition
 * @returns 
 */
export type SetBlurFunc = (st: boolean) => void;

type BackgroundBlurProps = {
    children: React.ReactNode
    setBlur: SetBlurFunc
}

/**
 * Creates a new background blur effect.
 * 
 * @returns A JSX element component
 */
export default function BackgroundBlur({children, setBlur}: BackgroundBlurProps): JSX.Element{
    const blurDivRef = useRef(null);

    useDisableBackgroundBlur(setBlur, blurDivRef);

    return (
        <div 
        className="w-screen h-screen bg-gray-400/50 backdrop-blur-[1px] absolute flex items-center justify-center"
        ref={blurDivRef}>
           {children} 
        </div>
    )
}
/**
 * A hook used to add listeners to disable the background blur.
 */
function useDisableBackgroundBlur(blurSetter: (st: boolean) => void, blurRef: RefObject<null | HTMLDivElement>){
    useEffect(() => {
        const keydownDisableBlur = (e: KeyboardEvent) => {
            if(e.key == "Escape"){
                blurSetter(false);
            }
        }
        const mouseDisableBlur = (e: MouseEvent) => {
            if(!blurRef.current){
                return;
            }

            if(blurRef.current == e.target){
                blurSetter(false);
            }
        }
        
        document.addEventListener("keydown", keydownDisableBlur);
        document.addEventListener("mousedown", mouseDisableBlur);

        return () => {
            document.removeEventListener("keydown", keydownDisableBlur);
            document.removeEventListener("mousedown", mouseDisableBlur);
        }
    }, [])
}