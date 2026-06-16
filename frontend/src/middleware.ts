import { createContext, redirect, type LoaderFunction, type MiddlewareFunction } from "react-router";
import type { ResponseApi } from "./response-types";
import { fetchApi } from "./functions/fetchtils";

export const userContext = createContext<User|null>(null);

export type User = {
    account_id: string
    username: string
    created_on: Date
    active: boolean
}

/**
 * Sends a request to the backend for authentication using the cookie
 * in the request header. This validates if the session ID is correct
 * and the user is valid.
 * 
 * Once valid, it will set the user context and it can be used for all routes
 * afterwards.
 * 
 * Authentication failures will redirect to the login page.
 */
export const authMiddleware: MiddlewareFunction = async ({context}) => {
    try{
        const user = await getUser();
        if(!user){
            throw redirect("/login");
        }

        context.set(userContext, user);
    }catch(err){
        throw err;
    }
}

/**
 * Sends a request to the backend for authentication using the cookie
 * in the request header. This validates if the session ID is correct
 * and the user is valid.
 * 
 * Upon a successful authentication it will redirect to the storage,
 * bypassing the login page.
 */
export const loginMiddleware: MiddlewareFunction = async () => {
    const res: ResponseApi<User> = await fetchApi("/api/user", "GET");

    if(res.status == "success"){
        throw redirect("/storage");
    }
}

export const getUserContext: LoaderFunction = async ({context}) => {
    const user = context.get(userContext);

    return user;
}

export async function getUser(): Promise<User|null>{
    const res: ResponseApi<User> = await fetchApi("/api/user", "GET");

    if (res.status == "error"){
        return null;
    }

    return res.output;
}