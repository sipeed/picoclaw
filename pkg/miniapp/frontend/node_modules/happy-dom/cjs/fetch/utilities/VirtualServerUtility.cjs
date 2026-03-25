"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const WindowBrowserContext_js_1 = __importDefault(require("../../window/WindowBrowserContext.cjs"));
const path_1 = __importDefault(require("path"));
const NOT_FOUND_HTML = '<html><head><title>Happy DOM Virtual Server - 404 Not Found</title></head><body><h1>Happy DOM Virtual Server - 404 Not Found</h1></body></html>';
/**
 * Virtual server utility.
 */
class VirtualServerUtility {
    /**
     * Returns the filesystem path for a request URL if it matches a virtual server.
     *
     * @param window Window.
     * @param requestURL Request URL.
     */
    static getFilepath(window, requestURL) {
        const browserSettings = new WindowBrowserContext_js_1.default(window).getSettings();
        if (!browserSettings || !browserSettings.fetch.virtualServers) {
            return null;
        }
        for (const virtualServer of browserSettings.fetch.virtualServers) {
            let baseURL;
            if (typeof virtualServer.url === 'string') {
                const url = new URL(virtualServer.url[virtualServer.url.length - 1] === '/'
                    ? virtualServer.url.slice(0, -1)
                    : virtualServer.url, window.location.origin);
                if (requestURL.startsWith(url.href)) {
                    baseURL = url;
                }
            }
            else if (virtualServer.url instanceof RegExp) {
                const match = requestURL.match(virtualServer.url);
                if (match) {
                    baseURL = new URL(match[0][match[0].length - 1] === '/' ? match[0].slice(0, -1) : match[0], window.location.origin);
                }
            }
            if (baseURL) {
                const path = requestURL.slice(baseURL.href.length).split('?')[0].split('#')[0];
                return path_1.default.join(path_1.default.resolve(virtualServer.directory), path.replaceAll('/', path_1.default.sep));
            }
        }
        return null;
    }
    /**
     * Returns a 404 response.
     *
     * @param window Window.
     * @returns 404 response.
     */
    static getNotFoundResponse(window) {
        return new window.Response(NOT_FOUND_HTML, {
            status: 404,
            statusText: 'Not Found',
            headers: {
                'Content-Type': 'text/html'
            }
        });
    }
    /**
     * Returns a 404 response.
     *
     * @param window Window.
     * @returns 404 response.
     */
    static getNotFoundSyncResponse(window) {
        return {
            status: 404,
            statusText: 'Not Found',
            ok: false,
            url: null,
            redirected: false,
            headers: new window.Headers({
                'Content-Type': 'text/html'
            }),
            body: Buffer.from(NOT_FOUND_HTML)
        };
    }
}
exports.default = VirtualServerUtility;
//# sourceMappingURL=VirtualServerUtility.cjs.map