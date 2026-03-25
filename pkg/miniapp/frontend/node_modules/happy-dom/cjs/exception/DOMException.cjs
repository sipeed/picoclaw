"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const DOMExceptionNameEnum_js_1 = __importDefault(require("./DOMExceptionNameEnum.cjs"));
/**
 * DOM Exception.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DOMException/DOMException.
 */
class DOMException extends Error {
    /**
     * Constructor.
     *
     * @param message Message.
     * @param name Name.
     */
    constructor(message, name = null) {
        super(message);
        this.name = name || DOMExceptionNameEnum_js_1.default.domException;
    }
}
exports.default = DOMException;
//# sourceMappingURL=DOMException.cjs.map