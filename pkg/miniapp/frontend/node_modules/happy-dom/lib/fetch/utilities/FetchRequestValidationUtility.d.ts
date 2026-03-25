import IRequestReferrerPolicy from '../types/IRequestReferrerPolicy.js';
import IRequestRedirect from '../types/IRequestRedirect.js';
import URL from '../../url/URL.js';
import Request from '../Request.js';
/**
 * Fetch request validation utility.
 */
export default class FetchRequestValidationUtility {
    /**
     * Validates request method.
     *
     * @throws DOMException
     * @param request Request.
     */
    static validateMethod(request: Request): void;
    /**
     * Validates request body.
     *
     * @throws DOMException
     * @param request Request.
     */
    static validateBody(request: Request): void;
    /**
     * Validates request URL.
     *
     * @throws DOMException
     * @param url URL.
     */
    static validateURL(url: URL): void;
    /**
     * Validates request referrer policy.
     *
     * @throws DOMException
     * @param referrerPolicy Referrer policy.
     */
    static validateReferrerPolicy(referrerPolicy: IRequestReferrerPolicy): void;
    /**
     * Validates request redirect.
     *
     * @throws DOMException
     * @param redirect Redirect.
     */
    static validateRedirect(redirect: IRequestRedirect): void;
    /**
     * Validates request redirect.
     *
     * @throws DOMException
     * @param request
     * @param redirect Redirect.
     */
    static validateSchema(request: Request): void;
}
//# sourceMappingURL=FetchRequestValidationUtility.d.ts.map