"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const UIEvent_js_1 = __importDefault(require("../UIEvent.cjs"));
/**
 *
 */
class ErrorEvent extends UIEvent_js_1.default {
    message;
    filename;
    lineno;
    colno;
    error;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.message = eventInit?.message ?? '';
        this.filename = eventInit?.filename ?? '';
        this.lineno = eventInit?.lineno ?? 0;
        this.colno = eventInit?.colno ?? 0;
        this.error = eventInit?.error ?? null;
    }
}
exports.default = ErrorEvent;
//# sourceMappingURL=ErrorEvent.cjs.map