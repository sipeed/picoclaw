"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
/**
 * HTML Opt Group Element.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLOptGroupElement
 */
class HTMLOptGroupElement extends HTMLElement_js_1.default {
    /**
     * Returns label.
     *
     * @returns Label.
     */
    get label() {
        return this.getAttribute('label') || '';
    }
    /**
     * Sets label.
     *
     * @param label Label.
     */
    set label(label) {
        if (!label) {
            this.removeAttribute('label');
        }
        else {
            this.setAttribute('label', label);
        }
    }
    /**
     * Returns disabled.
     *
     * @returns Disabled.
     */
    get disabled() {
        return this.getAttribute('disabled') !== null;
    }
    /**
     * Sets disabled.
     *
     * @param disabled Disabled.
     */
    set disabled(disabled) {
        if (!disabled) {
            this.removeAttribute('disabled');
        }
        else {
            this.setAttribute('disabled', '');
        }
    }
}
exports.default = HTMLOptGroupElement;
//# sourceMappingURL=HTMLOptGroupElement.cjs.map