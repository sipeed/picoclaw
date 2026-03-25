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
const Event_js_1 = __importDefault(require("../../event/Event.cjs"));
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * HTML Dialog Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLDialogElement
 */
class HTMLDialogElement extends HTMLElement_js_1.default {
    // Internal properties
    [PropertySymbol.returnValue] = '';
    // Events
    oncancel = null;
    onclose = null;
    /**
     * Returns return value.
     *
     * @returns Return value.
     */
    get returnValue() {
        return this[PropertySymbol.returnValue];
    }
    /**
     * Sets return value.
     *
     * @param value Return value.
     */
    set returnValue(value) {
        this[PropertySymbol.returnValue] = value;
    }
    /**
     * Sets the "open" attribute.
     *
     * @param open Open.
     */
    set open(open) {
        if (open) {
            this.setAttribute('open', '');
        }
        else {
            this.removeAttribute('open');
        }
    }
    /**
     * Returns open.
     *
     * @returns Open.
     */
    get open() {
        return this.getAttribute('open') !== null;
    }
    /**
     * Closes the dialog.
     *
     * @param [returnValue] ReturnValue.
     */
    close(returnValue = '') {
        const wasOpen = this.open;
        this.removeAttribute('open');
        this.returnValue = returnValue;
        if (wasOpen) {
            this.dispatchEvent(new Event_js_1.default('close'));
        }
    }
    /**
     * Shows the modal.
     */
    showModal() {
        this.setAttribute('open', '');
    }
    /**
     * Shows the dialog.
     */
    show() {
        this.setAttribute('open', '');
    }
}
exports.default = HTMLDialogElement;
//# sourceMappingURL=HTMLDialogElement.cjs.map