"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const UIEvent_js_1 = __importDefault(require("../UIEvent.cjs"));
/**
 *
 */
class TouchEvent extends UIEvent_js_1.default {
    altKey;
    changedTouches;
    ctrlKey;
    metaKey;
    shiftKey;
    targetTouches;
    touches;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.altKey = eventInit?.altKey ?? false;
        this.changedTouches = eventInit?.changedTouches ?? [];
        this.ctrlKey = eventInit?.ctrlKey ?? false;
        this.metaKey = eventInit?.metaKey ?? false;
        this.shiftKey = eventInit?.shiftKey ?? false;
        this.targetTouches = eventInit?.targetTouches ?? [];
        this.touches = eventInit?.touches ?? [];
    }
}
exports.default = TouchEvent;
//# sourceMappingURL=TouchEvent.cjs.map