import ICachedPreflightResponse from './ICachedPreflightResponse.cjs';
import ICachablePreflightRequest from './ICachablePreflightRequest.cjs';
import ICachablePreflightResponse from './ICachablePreflightResponse.cjs';
/**
 * Fetch response cache.
 */
export default interface IPreflightResponseCache {
    /**
     * Returns cached response.
     *
     * @param request Request.
     * @returns Cached response.
     */
    get(request: ICachablePreflightRequest): ICachedPreflightResponse | null;
    /**
     * Adds a cached response.
     *
     * @param request Request.
     * @param response Response.
     * @returns Cached response.
     */
    add(request: ICachablePreflightRequest, response: ICachablePreflightResponse): ICachedPreflightResponse | null;
    /**
     * Clears the cache.
     */
    clear(): void;
}
//# sourceMappingURL=IPreflightResponseCache.d.ts.map