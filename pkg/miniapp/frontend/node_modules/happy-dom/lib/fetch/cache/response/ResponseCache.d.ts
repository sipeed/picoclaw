import IResponseCache from './IResponseCache.js';
import ICachedResponse from './ICachedResponse.js';
import ICachableRequest from './ICachableRequest.js';
import ICachableResponse from './ICachableResponse.js';
/**
 * Fetch response cache.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching
 * @see https://www.mnot.net/cache_docs/
 */
export default class ResponseCache implements IResponseCache {
    #private;
    /**
     * Returns cached response.
     *
     * @param request Request.
     * @returns Cached response.
     */
    get(request: ICachableRequest): ICachedResponse | null;
    /**
     * Adds a cache entity.
     *
     * @param request Request.
     * @param response Response.
     * @returns Cached response.
     */
    add(request: ICachableRequest, response: ICachableResponse): ICachedResponse;
    /**
     * Clears the cache.
     *
     * @param [options] Options.
     * @param [options.url] URL.
     * @param [options.toTime] Removes all entries that are older than this time. Time in MS.
     */
    clear(options?: {
        url?: string;
        toTime?: number;
    }): void;
}
//# sourceMappingURL=ResponseCache.d.ts.map