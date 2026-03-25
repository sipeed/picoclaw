"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * Data URI parser.
 *
 * Based on:
 * https://github.com/TooTallNate/node-data-uri-to-buffer/blob/main/src/index.ts (MIT)
 */
class DataURIParser {
    /**
     * Returns a Buffer instance from the given data URI `uri`.
     *
     * @param uri Data URI.
     * @returns Buffer.
     */
    static parse(uri) {
        if (!/^data:/i.test(uri)) {
            throw new TypeError('`uri` does not appear to be a Data URI (must begin with "data:")');
        }
        // Strip newlines
        uri = uri.replace(/\r?\n/g, '');
        // Split the URI up into the "metadata" and the "data" portions
        const firstComma = uri.indexOf(',');
        if (firstComma === -1 || firstComma <= 4) {
            throw new TypeError('malformed data: URI');
        }
        // Remove the "data:" scheme and parse the metadata
        const meta = uri.substring(5, firstComma).split(';');
        let charset = '';
        let base64 = false;
        let type = meta[0] || 'text/plain';
        for (let i = 1; i < meta.length; i++) {
            if (meta[i] === 'base64') {
                base64 = true;
            }
            else if (meta[i]) {
                type += `;${meta[i]}`;
                if (meta[i].indexOf('charset=') === 0) {
                    charset = meta[i].substring(8);
                }
            }
        }
        // Defaults to US-ASCII only if type is not provided
        if (!meta[0] && !charset.length) {
            type += ';charset=US-ASCII';
            charset = 'US-ASCII';
        }
        // Get the encoded data portion and decode URI-encoded chars
        const encoding = base64 ? 'base64' : 'ascii';
        const data = unescape(uri.substring(firstComma + 1));
        const buffer = Buffer.from(data, encoding);
        return {
            type,
            charset,
            buffer
        };
    }
}
exports.default = DataURIParser;
//# sourceMappingURL=DataURIParser.cjs.map