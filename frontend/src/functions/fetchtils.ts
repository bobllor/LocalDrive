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
 * An error can occur in the call and must be caught.
 * 
 * @param path The non-base request URL, this can include the forward slash
 * @param method The method used on the request, by default it uses GET
 * @param data Any object used in the body
 * @param headers The headers with the request
 * @returns ResponseApi promise of type T
 */
export async function fetchApi<T>(path: string, method: Method = "GET", data?: {}, headers?: {}): Promise<ResponseApi<T>>{
    let body;

    if(data instanceof Blob){
        body = data;
    }else if(data !== undefined){
        body = JSON.stringify(data);
    }

    const res = await fetch(createUrl(path), {
        method: method,
        body: body,
        headers: headers,
        credentials: "include",
    });

    const resHeaders: Array<string> = [];
    res.headers.forEach((v, k) => {
        resHeaders.push(`${k}: ${v}`);
    })

    console.debug(`Response Headers: "${headers}" | Response type: ${res.type}`);
    const r: ResponseApi<T> = await res.json();

    // TODO: proper log, output is not logged
    console.debug(`Response status: ${r.status} | OK: ${res.ok}`);

    return r;
}

export type Method = "GET" | "POST" | "PUT" | "DELETE";