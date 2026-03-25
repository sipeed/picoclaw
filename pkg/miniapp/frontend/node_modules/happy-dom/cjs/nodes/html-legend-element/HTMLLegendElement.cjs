"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const HTMLFieldSetElement_js_1 = __importDefault(require("../html-field-set-element/HTMLFieldSetElement.cjs"));
/**
 * HTMLLegendElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLLegendElement
 */
class HTMLLegendElement extends HTMLElement_js_1.default {
    /**
     * Returns the parent form element.
     *
     * @returns Form.
     */
    get form() {
        let parent = this;
        while (parent) {
            if (parent instanceof HTMLFieldSetElement_js_1.default) {
                return parent.form;
            }
            parent = parent.parentNode;
        }
        return null;
    }
}
exports.default = HTMLLegendElement;
//# sourceMappingURL=HTMLLegendElement.cjs.map