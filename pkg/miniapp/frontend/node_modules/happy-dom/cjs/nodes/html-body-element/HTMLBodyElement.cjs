"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const HTMLElement_js_1 = __importDefault(require("../html-element/HTMLElement.cjs"));
/**
 * HTMLBodyElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLBodyElement
 */
class HTMLBodyElement extends HTMLElement_js_1.default {
    // Events
    onafterprint = null;
    onbeforeprint = null;
    onbeforeunload = null;
    ongamepadconnected = null;
    ongamepaddisconnected = null;
    onhashchange = null;
    onlanguagechange = null;
    onmessage = null;
    onmessageerror = null;
    onoffline = null;
    ononline = null;
    onpagehide = null;
    onpageshow = null;
    onpopstate = null;
    onrejectionhandled = null;
    onstorage = null;
    onunhandledrejection = null;
    onunload = null;
}
exports.default = HTMLBodyElement;
//# sourceMappingURL=HTMLBodyElement.cjs.map