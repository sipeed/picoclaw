import * as PropertySymbol from '../PropertySymbol.cjs';
import IHeadersInit from './types/IHeadersInit.cjs';
/**
 * Fetch headers.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Headers
 */
export default class Headers {
    [PropertySymbol.entries]: {
        [k: string]: {
            name: string;
            value: string[];
        };
    };
    /**
     * Constructor.
     *
     * @param init Headers init.
     */
    constructor(init?: IHeadersInit);
    /**
     * Appends a new value onto an existing header inside a Headers object, or adds the header if it does not already exist.
     *
     * @param name Name.
     * @param value Value.
     */
    append(name: string, value: string): void;
    /**
     * Removes an header.
     *
     * @param name Name.
     */
    delete(name: string): void;
    /**
     * Returns header value.
     *
     * @param name Name.
     * @returns Value.
     */
    get(name: string): string | null;
    /**
     * Sets a new value for an existing header inside a Headers object, or adds the header if it does not already exist.
     *
     * @param name Name.
     * @param value Value.
     */
    set(name: string, value: string): void;
    /**
     * Returns an array containing the values of all Set-Cookie headers associated with a response.
     *
     * @returns An array of strings representing the values of all the different Set-Cookie headers.
     */
    getSetCookie(): string[];
    /**
     * Returns whether an Headers object contains a certain key.
     *
     * @param name Name.
     * @returns "true" if the Headers object contains the key.
     */
    has(name: string): boolean;
    /**
     * Executes a callback function once per each key/value pair in the Headers object.
     *
     * @param callback Callback.
     */
    forEach(callback: (name: string, value: string, thisArg: Headers) => void): void;
    /**
     * Returns an iterator, allowing you to go through all keys of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    keys(): IterableIterator<string>;
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    values(): IterableIterator<string>;
    /**
     * Returns an iterator, allowing you to go through all key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    entries(): IterableIterator<[string, string]>;
    /**
     * Iterator.
     *
     * @returns Iterator.
     */
    [Symbol.iterator](): IterableIterator<[string, string]>;
}
//# sourceMappingURL=Headers.d.ts.map