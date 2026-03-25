import DOMException from '../exception/DOMException.js';
import * as PropertySymbol from '../PropertySymbol.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
/**
 * Fetch headers.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Headers
 */
export default class Headers {
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
                        throw new DOMException('Failed to construct "Headers": The provided init is not a valid array.', DOMExceptionNameEnum.invalidStateError);
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
//# sourceMappingURL=Headers.js.map