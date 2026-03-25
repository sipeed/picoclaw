import IPreflightResponseCache from './IPreflightResponseCache.cjs';
import ICachablePreflightRequest from './ICachablePreflightRequest.cjs';
import ICachedPreflightResponse from './ICachedPreflightResponse.cjs';
import ICachablePreflightResponse from './ICachablePreflightResponse.cjs';
/**
 * Fetch preflight response cache.
 *
 * @see https://developer.mozilla.org/en-US/docs/Glossary/Preflight_request
 */
export default class PreflightResponseCache implements IPreflightResponseCache {
    #private;
    /**
     * Returns cached response.
     *
     * @param request Request.
     * @returns Cached response.
     */
    get(request: ICachablePreflightRequest): ICachedPreflightResponse | null;
    /**
     * Adds a cache entity.
     *
     * @param request Request.
     * @param response Response.
     * @returns Cached response.
     */
    add(request: ICachablePreflightRequest, response: ICachablePreflightResponse): ICachedPreflightResponse;
    /**
     * Clears the cache.
     */
    clear(): void;
}
//# sourceMappingURL=PreflightResponseCache.d.ts.map