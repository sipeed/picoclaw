import URL from '../../url/URL.js';
import BrowserWindow from '../../window/BrowserWindow.js';
import Headers from '../Headers.js';
import IRequestReferrerPolicy from '../types/IRequestReferrerPolicy.js';
import Request from '../Request.js';
/**
 * Fetch referrer utility.
 */
export default class FetchRequestReferrerUtility {
    /**
     * Prepares the request before being sent.
     *
     * @param originURL Origin URL.
     * @param request Request.
     */
    static prepareRequest(originURL: URL, request: Request): void;
    /**
     * Returns initial referrer.
     *
     * @param window Window.
     * @param referrer Referrer.
     * @returns Initial referrer.
     */
    static getInitialReferrer(window: BrowserWindow, referrer: '' | 'no-referrer' | 'client' | string | URL): '' | 'no-referrer' | 'client' | URL;
    /**
     * Returns referrer policy from header.
     *
     * @see https://w3c.github.io/webappsec-referrer-policy/#parse-referrer-policy-from-header
     * @param headers Response headers
     * @returns Policy.
     */
    static getReferrerPolicyFromHeader(headers: Headers): IRequestReferrerPolicy;
    /**
     * Returns the request referrer to be used as the value for the "Referer" header.
     *
     * Based on:
     * https://github.com/node-fetch/node-fetch/blob/main/src/utils/referrer.js (MIT)
     *
     * @see https://w3c.github.io/webappsec-referrer-policy/#determine-requests-referrer
     * @param originURL Origin URL.
     * @param request Request.
     * @returns Request referrer.
     */
    private static getSentReferrer;
    /**
     * Returns "true" if the request's referrer is potentially trustworthy.
     *
     * @see https://w3c.github.io/webappsec-secure-contexts/#is-origin-trustworthy
     * @param url URL.
     * @returns "true" if the request's referrer is potentially trustworthy.
     */
    private static isURLPotentiallyTrustWorthy;
    /**
     * Returns "true" if the request's referrer origin is potentially trustworthy.
     *
     * @see https://w3c.github.io/webappsec-secure-contexts/#is-origin-trustworthy
     * @param url URL.
     * @returns "true" if the request's referrer origin is potentially trustworthy.
     */
    private static isOriginPotentiallyTrustWorthy;
}
//# sourceMappingURL=FetchRequestReferrerUtility.d.ts.map