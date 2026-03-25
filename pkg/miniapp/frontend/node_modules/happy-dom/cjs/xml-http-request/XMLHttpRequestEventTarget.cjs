"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const EventTarget_js_1 = __importDefault(require("../event/EventTarget.cjs"));
/**
 * References: https://xhr.spec.whatwg.org/#xmlhttprequesteventtarget.
 */
class XMLHttpRequestEventTarget extends EventTarget_js_1.default {
    onloadstart = null;
    onprogress = null;
    onabort = null;
    onerror = null;
    onload = null;
    ontimeout = null;
    onloadend = null;
}
exports.default = XMLHttpRequestEventTarget;
//# sourceMappingURL=XMLHttpRequestEventTarget.cjs.map