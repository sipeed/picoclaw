"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const BASE64_CHARS = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/=';
/**
 * Base64 encoding and decoding.
 */
class Base64 {
    /**
     * Creates a Base64-encoded ASCII string from a binary string (i.e., a string in which each character in the string is treated as a byte of binary data).
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/btoa
     * @param data Binay data.
     * @returns Base64-encoded string.
     */
    static btoa(data) {
        const str = data.toString();
        if (/[^\u0000-\u00ff]/.test(str)) {
            throw new DOMException_js_1.default("Failed to execute 'btoa' on 'Window': The string to be encoded contains characters outside of the Latin1 range.", DOMExceptionNameEnum_js_1.default.invalidCharacterError);
        }
        let t = '';
        let p = -6;
        let a = 0;
        let i = 0;
        let v = 0;
        let c;
        while (i < str.length || p > -6) {
            if (p < 0) {
                if (i < str.length) {
                    c = str.charCodeAt(i++);
                    v += 8;
                }
                else {
                    c = 0;
                }
                a = ((a & 255) << 8) | (c & 255);
                p += 8;
            }
            t += BASE64_CHARS.charAt(v > 0 ? (a >> p) & 63 : 64);
            p -= 6;
            v -= 6;
        }
        return t;
    }
    /**
     * Decodes a string of data which has been encoded using Base64 encoding.
     *
     * @see https://developer.mozilla.org/en-US/docs/Web/API/atob
     * @see https://infra.spec.whatwg.org/#forgiving-base64-encode.
     * @see Https://html.spec.whatwg.org/multipage/webappapis.html#btoa.
     * @param data Binay string.
     * @returns An ASCII string containing decoded data from encodedData.
     */
    static atob(data) {
        const str = data.toString();
        if (/[^\u0000-\u00ff]/.test(str)) {
            throw new DOMException_js_1.default("Failed to execute 'atob' on 'Window': The string to be decoded contains characters outside of the Latin1 range.", DOMExceptionNameEnum_js_1.default.invalidCharacterError);
        }
        if (/[^A-Za-z\d+/=]/.test(str) || str.length % 4 == 1) {
            throw new DOMException_js_1.default("Failed to execute 'atob' on 'Window': The string to be decoded is not correctly encoded.", DOMExceptionNameEnum_js_1.default.invalidCharacterError);
        }
        let t = '';
        let p = -8;
        let a = 0;
        let c;
        let d;
        for (let i = 0; i < str.length; i++) {
            if ((c = BASE64_CHARS.indexOf(str.charAt(i))) < 0) {
                continue;
            }
            a = (a << 6) | (c & 63);
            if ((p += 6) >= 0) {
                d = (a >> p) & 255;
                if (c !== 64) {
                    t += String.fromCharCode(d);
                }
                a &= 63;
                p -= 8;
            }
        }
        return t;
    }
}
exports.default = Base64;
//# sourceMappingURL=Base64.cjs.map