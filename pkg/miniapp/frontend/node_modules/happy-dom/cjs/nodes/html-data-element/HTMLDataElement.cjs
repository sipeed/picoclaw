"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
/**
 * HTMLDataElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDataElement
 */
class HTMLDataElement extends HTMLElement_js_1.default {
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        return this.getAttribute('value') || '';
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        this.setAttribute('value', value);
    }
}
exports.default = HTMLDataElement;
//# sourceMappingURL=HTMLDataElement.cjs.map