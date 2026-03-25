"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
/**
 * HTMLOListElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLOListElement
 */
class HTMLOListElement extends HTMLElement_js_1.default {
    /**
     * Returns reversed.
     *
     * @returns Reversed.
     */
    get reversed() {
        return this.getAttribute('reversed') !== null;
    }
    /**
     * Sets reversed.
     *
     * @param reversed Reversed.
     */
    set reversed(reversed) {
        if (!reversed) {
            this.removeAttribute('reversed');
        }
        else {
            this.setAttribute('reversed', '');
        }
    }
    /**
     * Returns start.
     *
     * @returns Start.
     */
    get start() {
        if (!this.hasAttribute('start')) {
            return 1;
        }
        const parsedValue = Number(this.getAttribute('start') || '');
        if (isNaN(parsedValue)) {
            return 1;
        }
        return parsedValue;
    }
    /**
     * Sets start.
     *
     * @param start Start.
     */
    set start(start) {
        if (typeof start !== 'number') {
            start = Number(start);
        }
        if (isNaN(start)) {
            start = 0;
        }
        this.setAttribute('start', String(start));
    }
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        return this.getAttribute('type') || '';
    }
    /**
     * Sets type.
     *
     * @param type Type.
     */
    set type(type) {
        this.setAttribute('type', type);
    }
}
exports.default = HTMLOListElement;
//# sourceMappingURL=HTMLOListElement.cjs.map