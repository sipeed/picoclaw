"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
/**
 * HTMLLIElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLLIElement
 */
class HTMLLIElement extends HTMLElement_js_1.default {
    /**
     * Returns value.
     *
     * @returns Value.
     */
    get value() {
        if (!this.hasAttribute('value')) {
            return 0;
        }
        const parsedValue = Number(this.getAttribute('value'));
        return isNaN(parsedValue) ? 0 : parsedValue;
    }
    /**
     * Sets value.
     *
     * @param value Value.
     */
    set value(value) {
        const parsedValue = Number(value);
        this.setAttribute('value', isNaN(parsedValue) ? '0' : String(parsedValue));
    }
}
exports.default = HTMLLIElement;
//# sourceMappingURL=HTMLLIElement.cjs.map