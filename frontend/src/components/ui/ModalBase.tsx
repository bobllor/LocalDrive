import type { JSX, ReactNode } from "react";
import type { AlignItems, JustifyContent } from "./ui-types";
import { ALIGN_ITEMS_MAP, JUSTIFY_CONTENT_MAP } from "./ui-vars";

type ModalBaseProps = {
    children: ReactNode
    justifyContent?: JustifyContent
    alignItems?: AlignItems
}

/**
 * The base modal component for modal popups. By default it will
 * align the items in the center.
 * 
 * It is not responsible for closing the modal or performing another
 * action.
 * 
 * @returns 
 */
export default function ModalBase({children, justifyContent = "center", alignItems = "center"}: ModalBaseProps): JSX.Element{
    return (
        <div className={`bg-white w-max-[90vw] h-max-[90vh] p-3
            flex ${JUSTIFY_CONTENT_MAP[justifyContent]} ${ALIGN_ITEMS_MAP[alignItems]}`}>
            {children}
        </div>
    )
}