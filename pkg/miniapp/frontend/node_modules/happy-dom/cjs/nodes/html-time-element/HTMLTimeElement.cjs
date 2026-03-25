"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
/**
 * HTMLTimeElement.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLTimeElement
 */
class HTMLTimeElement extends HTMLElement_js_1.default {
    /**
     * Returns dateTime.
     *
     * @returns dateTime.
     */
    get dateTime() {
        return this.getAttribute('dateTime') || '';
    }
    /**
     * Sets dateTime.
     *
     * @param dateTime dateTime.
     */
    set dateTime(dateTime) {
        this.setAttribute('dateTime', dateTime);
    }
}
exports.default = HTMLTimeElement;
//# sourceMappingURL=HTMLTimeElement.cjs.map