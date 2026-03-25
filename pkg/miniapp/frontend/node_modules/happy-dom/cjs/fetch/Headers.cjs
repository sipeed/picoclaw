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
const DOMException_js_1 = __importDefault(require("../exception/DOMException.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const DOMExceptionNameEnum_js_1 = __importDefault(require("../exception/DOMExceptionNameEnum.cjs"));
/**
 * Fetch headers.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Headers
 */
class Headers {
    [PropertySymbol.entries] = {};
    /**
     * Constructor.
     *
     * @param init Headers init.
     */
    constructor(init) {
        if (init) {
            if (init instanceof Headers) {
                this[PropertySymbol.entries] = JSON.parse(JSON.stringify(init[PropertySymbol.entries]));
            }
            else if (Array.isArray(init)) {
                for (const entry of init) {
                    if (entry.length !== 2) {
                        throw new DOMException_js_1.default('Failed to construct "Headers": The provided init is not a valid array.', DOMExceptionNameEnum_js_1.default.invalidStateError);
                    }
                    this.append(entry[0], entry[1]);
                }
            }
            else {
                for (const name of Object.keys(init)) {
                    this.set(name, init[name]);
                }
            }
        }
    }
    /**
     * Appends a new value onto an existing header inside a Headers object, or adds the header if it does not already exist.
     *
     * @param name Name.
     * @param value Value.
     */
    append(name, value) {
        const lowerName = name.toLowerCase();
        if (this[PropertySymbol.entries][lowerName]) {
            this[PropertySymbol.entries][lowerName].value.push(value);
        }
        else {
            this[PropertySymbol.entries][lowerName] = {
                name,
                value: [value]
            };
        }
    }
    /**
     * Removes an header.
     *
     * @param name Name.
     */
    delete(name) {
        delete this[PropertySymbol.entries][name.toLowerCase()];
    }
    /**
     * Returns header value.
     *
     * @param name Name.
     * @returns Value.
     */
    get(name) {
        return this[PropertySymbol.entries][name.toLowerCase()]?.value.join(', ') ?? null;
    }
    /**
     * Sets a new value for an existing header inside a Headers object, or adds the header if it does not already exist.
     *
     * @param name Name.
     * @param value Value.
     */
    set(name, value) {
        this[PropertySymbol.entries][name.toLowerCase()] = {
            name,
            value: [value]
        };
    }
    /**
     * Returns an array containing the values of all Set-Cookie headers associated with a response.
     *
     * @returns An array of strings representing the values of all the different Set-Cookie headers.
     */
    getSetCookie() {
        const entry = this[PropertySymbol.entries]['set-cookie'];
        if (!entry) {
            return [];
        }
        return entry.value;
    }
    /**
     * Returns whether an Headers object contains a certain key.
     *
     * @param name Name.
     * @returns "true" if the Headers object contains the key.
     */
    has(name) {
        return !!this[PropertySymbol.entries][name.toLowerCase()];
    }
    /**
     * Executes a callback function once per each key/value pair in the Headers object.
     *
     * @param callback Callback.
     */
    forEach(callback) {
        for (const header of Object.values(this[PropertySymbol.entries])) {
            callback(header.value.join(', '), header.name, this);
        }
    }
    /**
     * Returns an iterator, allowing you to go through all keys of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    *keys() {
        for (const header of Object.values(this[PropertySymbol.entries])) {
            yield header.name;
        }
    }
    /**
     * Returns an iterator, allowing you to go through all values of the key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    *values() {
        for (const header of Object.values(this[PropertySymbol.entries])) {
            yield header.value.join(', ');
        }
    }
    /**
     * Returns an iterator, allowing you to go through all key/value pairs contained in this object.
     *
     * @returns Iterator.
     */
    *entries() {
        for (const header of Object.values(this[PropertySymbol.entries])) {
            yield [header.name, header.value.join(', ')];
        }
    }
    /**
     * Iterator.
     *
     * @returns Iterator.
     */
    *[Symbol.iterator]() {
        for (const header of Object.values(this[PropertySymbol.entries])) {
            yield [header.name, header.value.join(', ')];
        }
    }
}
exports.default = Headers;
//# sourceMappingURL=Headers.cjs.map