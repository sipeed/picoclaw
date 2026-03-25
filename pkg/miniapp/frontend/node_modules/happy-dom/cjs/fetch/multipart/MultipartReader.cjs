"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DOMExceptionNameEnum_js_1 = __importDefault(require("../../exception/DOMExceptionNameEnum.cjs"));
const File_js_1 = __importDefault(require("../../file/File.cjs"));
var MultiparParserStateEnum;
(function (MultiparParserStateEnum) {
    MultiparParserStateEnum[MultiparParserStateEnum["boundary"] = 0] = "boundary";
    MultiparParserStateEnum[MultiparParserStateEnum["headerStart"] = 2] = "headerStart";
    MultiparParserStateEnum[MultiparParserStateEnum["header"] = 3] = "header";
    MultiparParserStateEnum[MultiparParserStateEnum["data"] = 5] = "data";
})(MultiparParserStateEnum || (MultiparParserStateEnum = {}));
const CHARACTER_CODE = {
    lf: 10,
    cr: 13
};
/**
 * Multipart reader.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/utils/multipart-parser.js (MIT)
 */
class MultipartReader {
    formData;
    boundary;
    boundaryIndex = 0;
    state = MultiparParserStateEnum.boundary;
    data = {
        contentDisposition: null,
        value: [],
        contentType: null,
        header: ''
    };
    window;
    /**
     * Constructor.
     *
     * @param window Window.
     * @param formData Form data.
     * @param boundary Boundary.
     */
    constructor(window, boundary) {
        const boundaryHeader = `--${boundary}`;
        this.window = window;
        this.boundary = new Uint8Array(boundaryHeader.length);
        this.formData = new window.FormData();
        for (let i = 0, max = boundaryHeader.length; i < max; i++) {
            this.boundary[i] = boundaryHeader.charCodeAt(i);
        }
    }
    /**
     * Appends data.
     *
     * @param data Data.
     */
    write(data) {
        let char;
        let nextChar;
        for (let i = 0, max = data.length; i < max; i++) {
            char = data[i];
            nextChar = data[i + 1];
            switch (this.state) {
                case MultiparParserStateEnum.boundary:
                    if (char === this.boundary[this.boundaryIndex]) {
                        this.boundaryIndex++;
                    }
                    else {
                        this.boundaryIndex = 0;
                    }
                    if (this.boundaryIndex === this.boundary.length) {
                        this.state = MultiparParserStateEnum.headerStart;
                        this.boundaryIndex = 0;
                    }
                    break;
                case MultiparParserStateEnum.headerStart:
                    if (nextChar !== CHARACTER_CODE.cr && nextChar !== CHARACTER_CODE.lf) {
                        this.data.header = '';
                        this.state =
                            data[i - 2] === CHARACTER_CODE.lf
                                ? MultiparParserStateEnum.data
                                : MultiparParserStateEnum.header;
                    }
                    break;
                case MultiparParserStateEnum.header:
                    if (char === CHARACTER_CODE.cr) {
                        if (this.data.header) {
                            const headerParts = this.data.header.split(':');
                            const headerName = headerParts[0].toLowerCase();
                            const headerValue = headerParts[1].trim();
                            switch (headerName) {
                                case 'content-disposition':
                                    this.data.contentDisposition = this.getContentDisposition(headerValue);
                                    break;
                                case 'content-type':
                                    this.data.contentType = headerValue;
                                    break;
                            }
                        }
                        this.state = MultiparParserStateEnum.headerStart;
                    }
                    else {
                        this.data.header += String.fromCharCode(char);
                    }
                    break;
                case MultiparParserStateEnum.data:
                    if (char === this.boundary[this.boundaryIndex]) {
                        this.boundaryIndex++;
                    }
                    else {
                        this.boundaryIndex = 0;
                    }
                    if (this.boundaryIndex === this.boundary.length) {
                        this.state = MultiparParserStateEnum.headerStart;
                        if (this.data.value.length) {
                            this.appendFormData(this.data.contentDisposition.name, Buffer.from(this.data.value.slice(0, -(this.boundary.length + 1))), this.data.contentDisposition.filename, this.data.contentType);
                            this.data.value = [];
                            this.data.contentDisposition = null;
                            this.data.contentType = null;
                        }
                        this.boundaryIndex = 0;
                    }
                    else {
                        this.data.value.push(char);
                    }
                    break;
            }
        }
    }
    /**
     * Ends the stream.
     *
     * @returns Form data.
     */
    end() {
        if (this.state !== MultiparParserStateEnum.data) {
            throw new this.window.DOMException(`Unexpected end of multipart stream. Expected state to be "${MultiparParserStateEnum.data}" but got "${this.state}".`, DOMExceptionNameEnum_js_1.default.invalidStateError);
        }
        this.appendFormData(this.data.contentDisposition.name, Buffer.from(this.data.value.slice(0, -2)), this.data.contentDisposition.filename, this.data.contentType);
        return this.formData;
    }
    /**
     * Appends data.
     *
     * @param key Key.
     * @param value value.
     * @param filename Filename.
     * @param type Type.
     */
    appendFormData(key, value, filename, type) {
        if (!value.length) {
            return;
        }
        if (filename) {
            this.formData.append(key, new File_js_1.default([value], filename, {
                type
            }));
        }
        else {
            this.formData.append(key, value.toString());
        }
    }
    /**
     * Returns content disposition.
     *
     * @param headerValue Header value.
     * @returns Content disposition.
     */
    getContentDisposition(headerValue) {
        const regex = /([a-z]+) *= *"([^"]+)"/g;
        const contentDisposition = {};
        let match;
        while ((match = regex.exec(headerValue))) {
            contentDisposition[match[1]] = match[2];
        }
        return contentDisposition;
    }
}
exports.default = MultipartReader;
//# sourceMappingURL=MultipartReader.cjs.map