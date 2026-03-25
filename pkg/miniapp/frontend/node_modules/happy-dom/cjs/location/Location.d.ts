import IBrowserFrame from '../browser/types/IBrowserFrame.cjs';
import * as PropertySymbol from '../PropertySymbol.cjs';
import { URL } from 'url';
/**
 * Location.
 */
export default class Location {
    #private;
    [Symbol.toStringTag]: string;
    /**
     * Constructor.
     *
     * @param browserFrame Browser frame.
     * @param url URL.
     */
    constructor(browserFrame: IBrowserFrame, url: string);
    /**
     * Returns hash.
     *
     * @returns Hash.
     */
    get hash(): string;
    /**
     * Sets hash.
     *
     * @param hash Value.
     */
    set hash(hash: string);
    /**
     * Returns host.
     *
     * @returns Host.
     */
    get host(): string;
    /**
     * Sets host.
     *
     * @param host Value.
     */
    set host(host: string);
    /**
     * Returns hostname.
     *
     * @returns Hostname.
     */
    get hostname(): string;
    /**
     * Sets hostname.
     *
     * @param hostname Value.
     */
    set hostname(hostname: string);
    /**
     * Override set href.
     */
    get href(): string;
    /**
     * Override set href.
     */
    set href(url: string);
    /**
     * Returns origin.
     *
     * @returns Origin.
     */
    get origin(): string;
    /**
     * Returns pathname
     *
     * @returns Pathname.
     */
    get pathname(): string;
    /**
     * Sets pathname.
     *
     * @param pathname Value.
     */
    set pathname(pathname: string);
    /**
     * Returns port.
     *
     * @returns Port.
     */
    get port(): string;
    /**
     * Sets port.
     *
     * @param port Value.
     */
    set port(port: string);
    /**
     * Returns protocol.
     *
     * @returns Protocol.
     */
    get protocol(): string;
    /**
     * Sets protocol.
     *
     * @param protocol Value.
     */
    set protocol(protocol: string);
    /**
     * Returns search.
     *
     * @returns Search.
     */
    get search(): string;
    /**
     * Sets search.
     *
     * @param search Value.
     */
    set search(search: string);
    /**
     * Replaces the current resource with the one at the provided URL. The difference from the assign() method is that after using replace() the current page will not be saved in session History, meaning the user won't be able to use the back button to navigate to it.
     *
     * @param url URL.
     */
    replace(url: string | URL): void;
    /**
     * Loads the resource at the URL provided in parameter.
     *
     * @param url URL.
     */
    assign(url: string | URL): void;
    /**
     * Reloads the resource from the current URL.
     */
    reload(): void;
    /**
     * Replaces the current URL state with the provided one without navigating to the new URL.
     *
     * @param browserFrame Browser frame that must match the current one as validation.
     * @param url URL.
     */
    [PropertySymbol.setURL](browserFrame: IBrowserFrame, url: string): void;
    /**
     * Destroys the location.
     */
    [PropertySymbol.destroy](): void;
    /**
     * Returns the URL as a string.
     *
     * @returns URL as a string.
     */
    toString(): string;
}
//# sourceMappingURL=Location.d.ts.map