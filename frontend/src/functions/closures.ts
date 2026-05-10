/**
 * A throttler wrapper for function execution.
 * @param f Any function
 * @param timeout The timeout before function is available to be used
 * @returns The function f wrapped in a throttle closure
 */
export function throttle(f: (...args: any) => any | Promise<any>, timeout: number = 600): (...args: any) => void{
    let flag = false;

    return function(...args: any){
        if(!flag){
            f(...args);
            flag = true;

            setTimeout(() => {
                flag = false;
            }, timeout);
        }
    }
}

/**
 * A debounce wrapper for function execution.
 * @param f Any function
 * @param timeout The time after a debounce before executing the given function
 * @returns The function f wrapped in a debounce closure
 */
export function debounce(f: (...args: any) => any | Promise<any>, timeout: number = 700): (...args: any) => void{
    let timer: number | undefined;

    return function(...args: any){
        clearTimeout(timer);

        timer = setTimeout(() => {
            f(...args);
        }, timeout)
    }
}