/**
 * Fetch preflight response cache.
 *
 * @see https://developer.mozilla.org/en-US/docs/Glossary/Preflight_request
 */
export default class PreflightResponseCache {
    #entries = {};
    /**
     * Returns cached response.
     *
     * @param request Request.
     * @returns Cached response.
     */
    get(request) {
        const cachedResponse = this.#entries[request.url];
        if (cachedResponse) {
            if (cachedResponse.expires < Date.now()) {
                delete this.#entries[request.url];
                return null;
            }
            return cachedResponse;
        }
        return null;
    }
    /**
     * Adds a cache entity.
     *
     * @param request Request.
     * @param response Response.
     * @returns Cached response.
     */
    add(request, response) {
        delete this.#entries[request.url];
        if (request.headers.get('Cache-Control')?.includes('no-cache')) {
            return null;
        }
        if (response.status < 200 || response.status >= 300) {
            return null;
        }
        const maxAge = response.headers.get('Access-Control-Max-Age');
        const allowOrigin = response.headers.get('Access-Control-Allow-Origin');
        if (!maxAge || !allowOrigin) {
            return null;
        }
        const allowMethods = [];
        if (response.headers.has('Access-Control-Allow-Methods')) {
            const allowMethodsHeader = response.headers.get('Access-Control-Allow-Methods');
            if (allowMethodsHeader !== '*') {
                for (const method of response.headers.get('Access-Control-Allow-Methods').split(',')) {
                    allowMethods.push(method.trim().toUpperCase());
                }
            }
        }
        const cachedResponse = {
            allowOrigin,
            allowMethods,
            expires: Date.now() + parseInt(maxAge) * 1000
        };
        if (isNaN(cachedResponse.expires) || cachedResponse.expires < Date.now()) {
            return null;
        }
        this.#entries[request.url] = cachedResponse;
        return cachedResponse;
    }
    /**
     * Clears the cache.
     */
    clear() {
        this.#entries = {};
    }
}
//# sourceMappingURL=PreflightResponseCache.js.map