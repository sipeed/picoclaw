import CachedResponseStateEnum from './CachedResponseStateEnum.cjs';
import Headers from '../../Headers.cjs';
export default interface ICachedResponse {
    /** Response. */
    response: {
        status: number;
        statusText: string;
        url: string;
        headers: Headers;
        waitingForBody: boolean;
        body: Buffer | null;
    };
    /** Request. */
    request: {
        headers: Headers;
        method: string;
    };
    /** Cache update time in milliseconds. */
    cacheUpdateTime: number;
    /** Last modified time in milliseconds. */
    lastModified: number | null;
    /** Vary headers. */
    vary: {
        [header: string]: string;
    };
    /** Expire time in milliseconds. */
    expires: number | null;
    /** ETag */
    etag: string | null;
    /** Must revalidate using "If-Modified-Since" request when expired. Not supported yet. */
    mustRevalidate: boolean;
    /** Stale while revalidate using "If-Modified-Since" request when expired */
    staleWhileRevalidate: boolean;
    /** Used when "mustRevalidate" or "staleWhileRevalidate" is true. */
    state: CachedResponseStateEnum;
}
//# sourceMappingURL=ICachedResponse.d.ts.map