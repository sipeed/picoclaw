import { Buffer } from 'buffer';
/**
 * Synchronous fetch script builder.
 */
export default class SyncFetchScriptBuilder {
    /**
     * Sends a synchronous request.
     *
     * @param request Request.
     * @param request.url
     * @param request.method
     * @param request.headers
     * @param request.body
     * @returns Script.
     */
    static getScript(request: {
        url: URL;
        method: string;
        headers: {
            [name: string]: string;
        };
        body?: Buffer | null;
    }): string;
}
//# sourceMappingURL=SyncFetchScriptBuilder.d.ts.map