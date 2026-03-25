"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const FetchHTTPSCertificate_js_1 = __importDefault(require("../certificate/FetchHTTPSCertificate.cjs"));
/**
 * Synchronous fetch script builder.
 */
class SyncFetchScriptBuilder {
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
    static getScript(request) {
        const sortedHeaders = {};
        const headerNames = Object.keys(request.headers).sort();
        for (const name of headerNames) {
            sortedHeaders[name] = request.headers[name];
        }
        return `
                const sendRequest = require('http${request.url.protocol === 'https:' ? 's' : ''}').request;
                const options = ${JSON.stringify({
            method: request.method,
            headers: sortedHeaders,
            agent: false,
            rejectUnauthorized: true,
            key: request.url.protocol === 'https:' ? FetchHTTPSCertificate_js_1.default.key : undefined,
            cert: request.url.protocol === 'https:' ? FetchHTTPSCertificate_js_1.default.cert : undefined
        }, null, 4)};
                const request = sendRequest(${JSON.stringify(request.url.href)}, options, (incomingMessage) => {
                    let data = Buffer.alloc(0);
                    incomingMessage.on('data', (chunk) => {
                        data = Buffer.concat([data, Buffer.from(chunk)]);
                    });
                    incomingMessage.on('end', () => {
                        console.log(JSON.stringify({
                            error: null,
                            incomingMessage: {
                                statusCode: incomingMessage.statusCode,
                                statusMessage: incomingMessage.statusMessage,
                                rawHeaders: incomingMessage.rawHeaders,
                                data: data.toString('base64')
                            }
                        }));
                    });
                    incomingMessage.on('error', (error) => {
                        console.log(JSON.stringify({ error: error.message, incomingMessage: null }));
                    });
                });
                request.write(Buffer.from('${request.body ? request.body.toString('base64') : ''}', 'base64'));
                request.end();
            `;
    }
}
exports.default = SyncFetchScriptBuilder;
//# sourceMappingURL=SyncFetchScriptBuilder.cjs.map