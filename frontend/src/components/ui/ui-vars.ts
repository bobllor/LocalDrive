import type { AlignItems, JustifyContent } from "./ui-types";

export const JUSTIFY_CONTENT_MAP: Record<JustifyContent, string> = {
    center: "justify-center",
    end: "justify-end",
    between: "justify-between",
    start: "justify-start",
}

export const ALIGN_ITEMS_MAP: Record<AlignItems, string> = {
    center: "items-center",
    end: "items-end",
    start: "items-start",
    baseline: "items-baseline",
}