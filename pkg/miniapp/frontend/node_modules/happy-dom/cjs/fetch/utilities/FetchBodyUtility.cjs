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
const MultipartFormDataParser_js_1 = __importDefault(require("../multipart/MultipartFormDataParser.cjs"));
const web_1 = require("stream/web");
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
const url_1 = require("url");
const FormData_js_1 = __importDefault(require("../../form-data/FormData.cjs"));
const Blob_js_1 = __importDefault(require("../../file/Blob.cjs"));
const DOMException_js_1 = __importDefault(require("../../exception/DOMException.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const buffer_1 = require("buffer");
const stream_1 = __importDefault(require("stream"));
/**
 * Fetch body utility.
 */
class FetchBodyUtility {
    /**
     * Parses body and returns stream and type.
     *
     * Based on:
     * https://github.com/node-fetch/node-fetch/blob/main/src/body.js (MIT)
     *
     * @param body Body.
     * @returns Stream and type.
     */
    static getBodyStream(body) {
        if (body === null || body === undefined) {
            return { stream: null, buffer: null, contentType: null, contentLength: null };
        }
        else if (body instanceof url_1.URLSearchParams) {
            const buffer = buffer_1.Buffer.from(body.toString());
            return {
                buffer,
                stream: this.toReadableStream(buffer),
                contentType: 'application/x-www-form-urlencoded;charset=UTF-8',
                contentLength: buffer.length
            };
        }
        else if (body instanceof Blob_js_1.default) {
            const buffer = body[PropertySymbol.buffer];
            return {
                buffer,
                stream: this.toReadableStream(buffer),
                contentType: body.type,
                contentLength: body.size
            };
        }
        else if (buffer_1.Buffer.isBuffer(body)) {
            return {
                buffer: body,
                stream: this.toReadableStream(body),
                contentType: null,
                contentLength: body.length
            };
        }
        else if (body instanceof ArrayBuffer) {
            const buffer = buffer_1.Buffer.from(body);
            return {
                buffer,
                stream: this.toReadableStream(buffer),
                contentType: null,
                contentLength: body.byteLength
            };
        }
        else if (ArrayBuffer.isView(body)) {
            const buffer = buffer_1.Buffer.from(body.buffer, body.byteOffset, body.byteLength);
            return {
                buffer,
                stream: this.toReadableStream(buffer),
                contentType: null,
                contentLength: body.byteLength
            };
        }
        else if (body instanceof web_1.ReadableStream) {
            return {
                buffer: null,
                stream: body,
                contentType: null,
                contentLength: null
            };
        }
        else if (body instanceof FormData_js_1.default) {
            return MultipartFormDataParser_js_1.default.formDataToStream(body);
        }
        const buffer = buffer_1.Buffer.from(String(body));
        return {
            buffer,
            stream: this.toReadableStream(buffer),
            contentType: 'text/plain;charset=UTF-8',
            contentLength: buffer.length
        };
    }
    /**
     * Clones a request or body body stream.
     *
     * It is actually not cloning the stream.
     * It creates a pass through stream and pipes the original stream to it.
     *
     * @param window Window.
     * @param requestOrResponse Request or Response.
     * @param requestOrResponse.body Body.
     * @param requestOrResponse.bodyUsed Body used.
     * @returns New stream.
     */
    static cloneBodyStream(window, requestOrResponse) {
        if (requestOrResponse.bodyUsed) {
            throw new window.DOMException(`Failed to clone body stream of request: Request body is already used.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        if (requestOrResponse.body === null || requestOrResponse.body === undefined) {
            return null;
        }
        // If a buffer is set, use it to create a new stream.
        if (requestOrResponse[PropertySymbol.buffer]) {
            return this.toReadableStream(requestOrResponse[PropertySymbol.buffer]);
        }
        // Pipe underlying node stream if it exists.
        if (requestOrResponse.body[PropertySymbol.nodeStream]) {
            const stream1 = new stream_1.default.PassThrough();
            const stream2 = new stream_1.default.PassThrough();
            requestOrResponse.body[PropertySymbol.nodeStream].pipe(stream1);
            requestOrResponse.body[PropertySymbol.nodeStream].pipe(stream2);
            // Sets the body of the cloned request/response to the first pass through stream.
            requestOrResponse.body = this.nodeToWebStream(stream1);
            // Returns the clone.
            return this.nodeToWebStream(stream2);
        }
        // Uses the tee() method to clone the ReadableStream
        // This requires the stream to be consumed in parallel which is not the case for the fetch API
        const [stream1, stream2] = requestOrResponse.body.tee();
        // Sets the body of the cloned request to the first pass through stream.
        // TODO: check id this is required as request should be read only object
        requestOrResponse.body == stream1;
        // Returns the other stream as the clone
        return stream2;
    }
    /**
     * Consume and convert an entire Body to a Buffer.
     *
     * Based on:
     * https://github.com/node-fetch/node-fetch/blob/main/src/body.js (MIT)
     *
     * @see https://fetch.spec.whatwg.org/#concept-body-consume-body
     * @param window Window.
     * @param body Body stream.
     * @returns Promise.
     */
    static async consumeBodyStream(window, body) {
        if (body === null || !(body instanceof web_1.ReadableStream)) {
            return buffer_1.Buffer.alloc(0);
        }
        if (body[PropertySymbol.error]) {
            throw body[PropertySymbol.error];
        }
        const reader = body.getReader();
        const chunks = [];
        let bytes = 0;
        try {
            let readResult = await reader.read();
            while (!readResult.done) {
                if (body[PropertySymbol.error]) {
                    throw body[PropertySymbol.error];
                }
                if (body[PropertySymbol.aborted]) {
                    throw new window.DOMException('Failed to read response body: The stream was aborted.', DOMExceptionNameEnum_js_1.default.abortError);
                }
                const chunk = readResult.value;
                bytes += chunk.length;
                chunks.push(chunk);
                readResult = await reader.read();
            }
        }
        catch (error) {
            if (error instanceof DOMException_js_1.default) {
                throw error;
            }
            throw new window.DOMException(`Failed to read response body. Error: ${error.message}.`, DOMExceptionNameEnum_js_1.default.encodingError);
        }
        try {
            if (typeof chunks[0] === 'string') {
                return buffer_1.Buffer.from(chunks.join(''));
            }
            return buffer_1.Buffer.concat(chunks, bytes);
        }
        catch (error) {
            throw new window.DOMException(`Could not create Buffer from response body. Error: ${error.message}.`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
    }
    /**
     * Wraps a given value in a browser ReadableStream.
     *
     * This method creates a ReadableStream and immediately enqueues and closes it
     * with the provided value, useful for stream API compatibility.
     *
     * @param value The value to be wrapped in a ReadableStream.
     * @returns ReadableStream
     */
    static toReadableStream(value) {
        return new web_1.ReadableStream({
            start(controller) {
                controller.enqueue(value);
                controller.close();
            }
        });
    }
    /**
     * Wraps a Node.js stream into a browser-compatible ReadableStream.
     *
     * Enables the use of Node.js streams where browser ReadableStreams are required.
     * Handles 'data', 'end', and 'error' events from the Node.js stream.
     *
     * @param nodeStream The Node.js stream to be converted.
     * @returns ReadableStream
     */
    static nodeToWebStream(nodeStream) {
        const readableStream = new web_1.ReadableStream({
            start(controller) {
                nodeStream.on('data', (chunk) => {
                    controller.enqueue(chunk);
                });
                nodeStream.on('end', () => {
                    controller.close();
                });
                nodeStream.on('error', (err) => {
                    controller.error(err);
                });
            }
        });
        readableStream[PropertySymbol.nodeStream] = nodeStream;
        return readableStream;
    }
}
exports.default = FetchBodyUtility;
//# sourceMappingURL=FetchBodyUtility.cjs.map