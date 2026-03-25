import WhatwgMIMEType from 'whatwg-mimetype';
import * as PropertySymbol from '../PropertySymbol.js';
import ProgressEvent from '../event/events/ProgressEvent.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import Blob from './Blob.js';
import FileReaderReadyStateEnum from './FileReaderReadyStateEnum.js';
import FileReaderFormatEnum from './FileReaderFormatEnum.js';
import EventTarget from '../event/EventTarget.js';
import FileReaderEventTypeEnum from './FileReaderEventTypeEnum.js';
import { Buffer } from 'buffer';
/**
 * Reference:
 * https://developer.mozilla.org/sv-SE/docs/Web/API/FileReader.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/file-api/FileReader-impl.js (MIT licensed).
 */
export default class FileReader extends EventTarget {
    error = null;
    result = null;
    readyState = FileReaderReadyStateEnum.empty;
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
        if (!(blob instanceof Blob)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'readAsArrayBuffer' on 'FileReader': parameter 1 is not of type 'Blob'.`);
        }
        this.#readFile(blob, FileReaderFormatEnum.buffer);
    }
    /**
     * Reads as binary string.
     *
     * @param blob Blob.
     */
    readAsBinaryString(blob) {
        if (!(blob instanceof Blob)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'readAsBinaryString' on 'FileReader': parameter 1 is not of type 'Blob'.`);
        }
        this.#readFile(blob, FileReaderFormatEnum.binaryString);
    }
    /**
     * Reads as data URL.
     *
     * @param blob Blob.
     */
    readAsDataURL(blob) {
        if (!(blob instanceof Blob)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'readAsDataURL' on 'FileReader': parameter 1 is not of type 'Blob'.`);
        }
        this.#readFile(blob, FileReaderFormatEnum.dataURL);
    }
    /**
     * Reads as text.
     *
     * @param blob Blob.
     * @param [encoding] Encoding.
     */
    readAsText(blob, encoding = null) {
        if (!(blob instanceof Blob)) {
            throw new this[PropertySymbol.window].TypeError(`Failed to execute 'readAsText' on 'FileReader': parameter 1 is not of type 'Blob'.`);
        }
        this.#readFile(blob, FileReaderFormatEnum.text, encoding || 'UTF-8');
    }
    /**
     * Aborts the file reader.
     */
    abort() {
        const window = this[PropertySymbol.window];
        window.clearTimeout(this.#loadTimeout);
        window.clearTimeout(this.#parseTimeout);
        if (this.readyState === FileReaderReadyStateEnum.empty ||
            this.readyState === FileReaderReadyStateEnum.done) {
            this.result = null;
            return;
        }
        if (this.readyState === FileReaderReadyStateEnum.loading) {
            this.readyState = FileReaderReadyStateEnum.done;
            this.result = null;
        }
        this.#isTerminated = true;
        this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.abort));
        this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.loadend));
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
        if (this.readyState === FileReaderReadyStateEnum.loading) {
            throw new window.DOMException('The object is in an invalid state.', DOMExceptionNameEnum.invalidStateError);
        }
        this.readyState = FileReaderReadyStateEnum.loading;
        this.#loadTimeout = window.setTimeout(() => {
            if (this.#isTerminated) {
                this.#isTerminated = false;
                return;
            }
            this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.loadstart));
            let data = blob[PropertySymbol.buffer];
            if (!data) {
                data = Buffer.alloc(0);
            }
            this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.loadstart, {
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
                    case FileReaderFormatEnum.buffer: {
                        this.result = new Uint8Array(data).buffer;
                        break;
                    }
                    case FileReaderFormatEnum.binaryString: {
                        this.result = data.toString('binary');
                        break;
                    }
                    case FileReaderFormatEnum.dataURL: {
                        // Spec seems very unclear here; see https://github.com/w3c/FileAPI/issues/104.
                        const contentType = WhatwgMIMEType.parse(blob.type) || 'application/octet-stream';
                        this.result =
                            `data:${contentType};base64,${data.toString('base64')}`;
                        break;
                    }
                    case FileReaderFormatEnum.text: {
                        this.result = new TextDecoder(encoding || 'UTF-8').decode(data);
                        break;
                    }
                }
                this.readyState = FileReaderReadyStateEnum.done;
                this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.load));
                this.dispatchEvent(new ProgressEvent(FileReaderEventTypeEnum.loadend));
            });
        });
    }
}
//# sourceMappingURL=FileReader.js.map