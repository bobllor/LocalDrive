import type { ResponseApi } from "../response-types";
import { createUrl } from "../server-utils";

/**
 * Sends a request to validate the current session. This sends
 * the cookie to the backend.
 * 
 * @returns The session validation status
 */
export async function validateSession(): Promise<boolean>{
    const res = await fetchApi<boolean>("/api/session");

    if(res.status == "error"){
        return false;
    }

    return res.output;
}

/**
 * Sends a request to the given path and returns its response.
 * If args are used, it will send the data with args.
 * 
 * An error can occur in the call, and must be caught.
 * 
 * @param path The non-base request URL, this can include the forward slash
 * @param method The method used on the request, by default it uses GET
 * @param body Any object that is being sent to the backend
 * @returns ResponseApi promise of type T
 */
export async function fetchApi<T>(path: string, method: Method = "GET", data?: {}): Promise<ResponseApi<T>>{
    const res = await fetch(createUrl(path), {
        method: method,
        body: !data ? undefined : JSON.stringify(data),
        credentials: "include",
    });

    const r: ResponseApi<T> = await res.json()

    // TODO: proper log, output is not logged
    console.debug(`Response status: ${r.status}`);
    if(r.status == "error"){
        throw r;
    }

    return r;
}

export type Method = "GET" | "POST" | "PUT" | "DELETE";