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
const whatwg_mimetype_1 = __importDefault(require("whatwg-mimetype"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const ProgressEvent_js_1 = __importDefault(require("../event/events/ProgressEvent.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
const Blob_js_1 = __importDefault(require("./Blob.cjs"));
const FileReaderReadyStateEnum_js_1 = __importDefault(require("./FileReaderReadyStateEnum.cjs"));
const FileReaderFormatEnum_js_1 = __importDefault(require("./FileReaderFormatEnum.cjs"));
const EventTarget_js_1 = __importDefault(require("../event/EventTarget.cjs"));
const FileReaderEventTypeEnum_js_1 = __importDefault(require("./FileReaderEventTypeEnum.cjs"));
const buffer_1 = require("buffer");
/**
 * Reference:
 * https://developer.mozilla.org/sv-SE/docs/Web/API/FileReader.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/file-api/FileReader-impl.js (MIT licensed).
 */
class FileReader extends EventTarget_js_1.default {
    error = null;
    result = null;
    readyState = FileReaderReadyStateEnum_js_1.default.empty;
    onabort = null;
    onerror = null;
    onload = null;
    onloadstart = null;
    onloadend = null;
    onprogress = null;
    #isTerminated = false;
    #loadTimeout = null;
    #parseTimeout = null;
    /**
     * Constructor.
     */
    constructor() {
        super();
        if (!this[PropertySymbol.window]) {
            throw new TypeError(`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`);
        }
    }
    /**
     * Reads as ArrayBuffer.
     *
     * @param blob Blob.
     */
    readAsArrayBuffer(blob) {
        if (!(blob instanceof Blob_js_1.default)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'readAsArrayBuffer' on 'FileReader': parameter 1 is not of type 'Blob'.`);
        }
        this.#readFile(blob, FileReaderFormatEnum_js_1.default.buffer);
    }
    /**
     * Reads as binary string.
     *
     * @param blob Blob.
     */
    readAsBinaryString(blob) {
        if (!(blob instanceof Blob_js_1.default)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'readAsBinaryString' on 'FileReader': parameter 1 is not of type 'Blob'.`);
        }
        this.#readFile(blob, FileReaderFormatEnum_js_1.default.binaryString);
    }
    /**
     * Reads as data URL.
     *
     * @param blob Blob.
     */
    readAsDataURL(blob) {
        if (!(blob instanceof Blob_js_1.default)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'readAsDataURL' on 'FileReader': parameter 1 is not of type 'Blob'.`);
        }
        this.#readFile(blob, FileReaderFormatEnum_js_1.default.dataURL);
    }
    /**
     * Reads as text.
     *
     * @param blob Blob.
     * @param [encoding] Encoding.
     */
    readAsText(blob, encoding = null) {
        if (!(blob instanceof Blob_js_1.default)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'readAsText' on 'FileReader': parameter 1 is not of type 'Blob'.`);
        }
        this.#readFile(blob, FileReaderFormatEnum_js_1.default.text, encoding || 'UTF-8');
    }
    /**
     * Aborts the file reader.
     */
    abort() {
        const window = this[PropertySymbol.window];
        window.clearTimeout(this.#loadTimeout);
        window.clearTimeout(this.#parseTimeout);
        if (this.readyState === FileReaderReadyStateEnum_js_1.default.empty ||
            this.readyState === FileReaderReadyStateEnum_js_1.default.done) {
            this.result = null;
            return;
        }
        if (this.readyState === FileReaderReadyStateEnum_js_1.default.loading) {
            this.readyState = FileReaderReadyStateEnum_js_1.default.done;
            this.result = null;
        }
        this.#isTerminated = true;
        this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.abort));
        this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.loadend));
    }
    /**
     * Reads a file.
     *
     * @param blob Blob.
     * @param format Format.
     * @param [encoding] Encoding.
     */
    #readFile(blob, format, encoding = null) {
        const window = this[PropertySymbol.window];
        if (this.readyState === FileReaderReadyStateEnum_js_1.default.loading) {
            throw new window.DOMException('The object is in an invalid state.', DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        this.readyState = FileReaderReadyStateEnum_js_1.default.loading;
        this.#loadTimeout = window.setTimeout(() => {
            if (this.#isTerminated) {
                this.#isTerminated = false;
                return;
            }
            this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.loadstart));
            let data = blob[PropertySymbol.buffer];
            if (!data) {
                data = buffer_1.Buffer.alloc(0);
            }
            this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.loadstart, {
                lengthComputable: !isNaN(blob.size),
                total: blob.size,
                loaded: data.length
            }));
            this.#parseTimeout = window.setTimeout(() => {
                if (this.#isTerminated) {
                    this.#isTerminated = false;
                    return;
                }
                switch (format) {
                    default:
                    case FileReaderFormatEnum_js_1.default.buffer: {
                        this.result = new Uint8Array(data).buffer;
                        break;
                    }
                    case FileReaderFormatEnum_js_1.default.binaryString: {
                        this.result = data.toString('binary');
                        break;
                    }
                    case FileReaderFormatEnum_js_1.default.dataURL: {
                        // Spec seems very unclear here; see https://github.com/w3c/FileAPI/issues/104.
                        const contentType = whatwg_mimetype_1.default.parse(blob.type) || 'application/octet-stream';
                        this.result =
                            `data:${contentType};base64,${data.toString('base64')}`;
                        break;
                    }
                    case FileReaderFormatEnum_js_1.default.text: {
                        this.result = new TextDecoder(encoding || 'UTF-8').decode(data);
                        break;
                    }
                }
                this.readyState = FileReaderReadyStateEnum_js_1.default.done;
                this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.load));
                this.dispatchEvent(new ProgressEvent_js_1.default(FileReaderEventTypeEnum_js_1.default.loadend));
            });
        });
    }
}
exports.default = FileReader;
//# sourceMappingURL=FileReader.cjs.map