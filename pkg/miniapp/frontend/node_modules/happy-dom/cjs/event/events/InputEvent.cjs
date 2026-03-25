"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const UIEvent_js_1 = __importDefault(require("../UIEvent.cjs"));
/**
 *
 */
class InputEvent extends UIEvent_js_1.default {
    data;
    dataTransfer;
    inputType;
    isComposing;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.data = eventInit?.data ?? '';
        this.dataTransfer = eventInit?.dataTransfer ?? null;
        this.inputType = eventInit?.inputType ?? '';
        this.isComposing = eventInit?.isComposing ?? false;
    }
}
exports.default = InputEvent;
//# sourceMappingURL=InputEvent.cjs.map