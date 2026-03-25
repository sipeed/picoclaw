"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const web_1 = require("stream/web");
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const MultipartReader_js_1 = __importDefault(require("./MultipartReader.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const buffer_1 = require("buffer");
/**
 * Multipart form data factory.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/utils/multipart-parser.js (MIT)
 */
class MultipartFormDataParser {
    /**
     * Returns form data.
     *
     * @param window Window.
     * @param body Body.
     * @param contentType Content type header value.
     * @returns Form data.
     */
    static async streamToFormData(window, body, contentType) {
        if (!/multipart/i.test(contentType)) {
            throw new window.DOMException(`Failed to build FormData object: The "content-type" header isn't of type "multipart/form-data".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        const match = contentType.match(/boundary=(?:"([^"]+)"|([^;]+))/i);
        if (!match) {
            throw new window.DOMException(`Failed to build FormData object: The "content-type" header doesn't contain any multipart boundary.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        const bodyReader = body.getReader();
        const reader = new MultipartReader_js_1.default(window, match[1] || match[2]);
        const chunks = [];
        let buffer;
        const bytes = 0;
        let readResult = await bodyReader.read();
        while (!readResult.done) {
            if (body[PropertySymbol.error]) {
                throw body[PropertySymbol.error];
            }
            if (body[PropertySymbol.aborted]) {
                throw new window.DOMException('Failed to read response body: The stream was aborted.', DOMExceptionNameEnum_js_1.default.abortError);
            }
            reader.write(readResult.value);
            readResult = await bodyReader.read();
        }
        try {
            buffer =
                typeof chunks[0] === 'string' ? buffer_1.Buffer.from(chunks.join('')) : buffer_1.Buffer.concat(chunks, bytes);
        }
        catch (error) {
            throw new window.DOMException(`Could not create Buffer from response body. Error: ${error.message}.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        return {
            formData: reader.end(),
            buffer
        };
    }
    /**
     * Converts a FormData object to a ReadableStream.
     *
     * @param formData FormData.
     * @returns Stream and type.
     */
    static formDataToStream(formData) {
        const boundary = '----HappyDOMFormDataBoundary' + Math.random().toString(36);
        const chunks = [];
        const prefix = `--${boundary}\r\nContent-Disposition: form-data; name="`;
        for (const [name, value] of formData) {
            if (typeof value === 'string') {
                chunks.push(buffer_1.Buffer.from(`${prefix}${this.escapeName(name)}"\r\n\r\n${value.replace(/\r(?!\n)|(?<!\r)\n/g, '\r\n')}\r\n`));
            }
            else {
                chunks.push(buffer_1.Buffer.from(`${prefix}${this.escapeName(name)}"; filename="${this.escapeName(value.name, true)}"\r\nContent-Type: ${value.type || 'application/octet-stream'}\r\n\r\n`));
                chunks.push(value[PropertySymbol.buffer]);
                chunks.push(buffer_1.Buffer.from('\r\n'));
            }
        }
        const buffer = buffer_1.Buffer.concat(chunks);
        return {
            contentType: `multipart/form-data; boundary=${boundary}`,
            contentLength: buffer.length,
            buffer,
            stream: new web_1.ReadableStream({
                start(controller) {
                    controller.enqueue(buffer);
                    controller.close();
                }
            })
        };
    }
    /**
     * Escapes a form data entry name.
     *
     * @param name Name.
     * @param filename Whether it is a filename.
     * @returns Escaped name.
     */
    static escapeName(name, filename = false) {
        return (filename ? name : name.replace(/\r?\n|\r/g, '\r\n'))
            .replace(/\n/g, '%0A')
            .replace(/\r/g, '%0D')
            .replace(/"/g, '%22');
    }
}
exports.default = MultipartFormDataParser;
//# sourceMappingURL=MultipartFormDataParser.cjs.map