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
const Blob_js_1 = __importDefault(require("../file/Blob.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const File_js_1 = __importDefault(require("../file/File.cjs"));
/**
 * FormData.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/FormData
 */
class FormData {
    #entries = [];
    /**
     * Constructor.
     *
     * @param [form] Form.
     */
    constructor(form) {
        if (!form) {
            return;
        }
        const items = form[PropertySymbol.getFormControlItems]();
        for (const item of items) {
            const name = item.name;
            if (name) {
                switch (item[PropertySymbol.tagName]) {
                    case 'INPUT':
                        switch (item.type) {
                            case 'file':
                                if (item[PropertySymbol.files].length === 0) {
                                    this.append(name, new File_js_1.default([], '', { type: 'application/octet-stream' }));
                                }
                                else {
                                    for (const file of item[PropertySymbol.files]) {
                                        this.append(name, file);
                                    }
                                }
                                break;
                            case 'checkbox':
                            case 'radio':
                                if (item.checked) {
                                    this.append(name, item.value);
                                }
                                break;
                            case 'submit':
                            case 'reset':
                            case 'button':
                                if (item.value) {
                                    this.append(name, item.value);
                                }
                                break;
                            default:
                                this.append(name, item.value);
                                break;
                        }
                        break;
                    case 'BUTTON':
                        if (item.value) {
                            this.append(name, item.value);
                        }
                        break;
                    case 'TEXTAREA':
                    case 'SELECT':
                        this.append(name, item.value);
                        break;
                }
            }
        }
    }
    /**
     * For each.
     *
     * @param callback Callback.
     */
    forEach(callback) {
        for (const entry of this.#entries) {
            callback.call(this, entry.value, entry.name, this);
        }
    }
    /**
     * Appends a new value onto an existing key.
     *
     * @param name Name.
     * @param value Value.
     * @param [filename] Filename.
     */
    append(name, value, filename) {
        if (filename && !(value instanceof Blob_js_1.default)) {
            throw new this[PropertySymbol.window].TypeError('Failed to execute "append" on "FormData": parameter 2 is not of type "Blob".');
        }
        this.#entries.push({
            name,
            value: this.#parseValue(value, filename)
        });
    }
    /**
     * Removes a value.
     *
     * @param name Name.
     */
    delete(name) {
        const newEntries = [];
        for (const entry of this.#entries) {
            if (entry.name !== name) {
                newEntries.push(entry);
            }
        }
        this.#entries = newEntries;
    }
    /**
     * Returns value.
     *
     * @param name Name.
     * @returns Value.
     */
    get(name) {
        for (const entry of this.#entries) {
            if (entry.name === name) {
                return entry.value;
            }
        }
        return null;
    }
    /**
     * Returns all values associated with the given name.
     *
     * @param name Name.
     * @returns Values.
     */
    getAll(name) {
        const values = [];
        for (const entry of this.#entries) {
            if (entry.name === name) {
                values.push(entry.value);
            }
        }
        return values;
    }
    /**
     * Returns whether a FormData object contains a certain key.
     *
     * @param name Name.
     * @returns "true" if the FormData object contains the key.
     */
    has(name) {
        for (const entry of this.#entries) {
            if (entry.name === name) {
                return true;
            }
        }
        return false;
    }
    /**
     * Sets a new value for an existing key inside a FormData object, or adds the key/value if it does not already exist.
     *
     * @param name Name.
     * @param value Value.
     * @param [filename] Filename.
     */
    set(name, value, filename) {
        for (const entry of this.#entries) {
            if (entry.name === name) {
                entry.value = this.#parseValue(value, filename);
                return;
            }
        }
        this.append(name, value);
    }
    /**
     * Returns an iterator, allowing you to go through all keys of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    *keys() {
        for (const entry of this.#entries) {
            yield entry.name;
        }
    }
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    *values() {
        for (const entry of this.#entries) {
            yield entry.value;
        }
    }
    /**
     * Returns an iterator, allowing you to go through all key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    *entries() {
        for (const entry of this.#entries) {
            yield [entry.name, entry.value];
        }
    }
    /**
     * Iterator.
     *
     * @returns Iterator.
     */
    *[Symbol.iterator]() {
        for (const entry of this.#entries) {
            yield [entry.name, entry.value];
        }
    }
    /**
     * Parses a value.
     *
     * @param value Value.
     * @param [filename] Filename.
     * @returns Parsed value.
     */
    #parseValue(value, filename) {
        if (value instanceof File_js_1.default) {
            if (filename) {
                const file = new File_js_1.default([], filename, { type: value.type, lastModified: value.lastModified });
                file[PropertySymbol.buffer] = value[PropertySymbol.buffer];
                return file;
            }
            return value;
        }
        if (value instanceof Blob_js_1.default) {
            const file = new File_js_1.default([], 'blob', { type: value.type });
            file[PropertySymbol.buffer] = value[PropertySymbol.buffer];
            return file;
        }
        return String(value);
    }
}
exports.default = FormData;
//# sourceMappingURL=FormData.cjs.map