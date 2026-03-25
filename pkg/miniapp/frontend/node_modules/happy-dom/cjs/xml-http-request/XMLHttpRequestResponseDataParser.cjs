"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const XMLHttpResponseTypeEnum_js_1 = __importDefault(require("./XMLHttpResponseTypeEnum.cjs"));
/**
 *
 */
class XMLHttpRequestResponseDataParser {
    /**
     * Parses response.
     *
     * @param options Options.
     * @param options.window Window.
     * @param [options.responseType] Response type.
     * @param [options.data] Data.
     * @param [options.contentType] Content type.
     * @returns Parsed response.
     **/
    static parse(options) {
        if (!options.data) {
            return '';
        }
        switch (options.responseType) {
            case XMLHttpResponseTypeEnum_js_1.default.arraybuffer:
                // See: https://github.com/jsdom/jsdom/blob/c3c421c364510e053478520500bccafd97f5fa39/lib/jsdom/living/helpers/binary-data.js
                const newAB = new ArrayBuffer(options.data.length);
                const view = new Uint8Array(newAB);
                view.set(options.data);
                return view;
            case XMLHttpResponseTypeEnum_js_1.default.blob:
                try {
                    return new options.window.Blob([new Uint8Array(options.data)], {
                        type: options.contentType || ''
                    });
                }
                catch (e) {
                    // Ignore error.
                }
                return null;
            case XMLHttpResponseTypeEnum_js_1.default.document:
                const window = options.window;
                const domParser = new window.DOMParser();
                try {
                    return domParser.parseFromString(options.data.toString(), 'application/xml');
                }
                catch (e) {
                    // Ignore error.
                }
                return null;
            case XMLHttpResponseTypeEnum_js_1.default.json:
                try {
                    return JSON.parse(options.data.toString());
                }
                catch (e) {
                    // Ignore error.
                }
                return null;
            case XMLHttpResponseTypeEnum_js_1.default.text:
            case '':
            default:
                return options.data.toString();
        }
    }
}
exports.default = XMLHttpRequestResponseDataParser;
//# sourceMappingURL=XMLHttpRequestResponseDataParser.cjs.map