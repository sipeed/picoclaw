"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const UIEvent_js_1 = __importDefault(require("../UIEvent.cjs"));
/**
 *
 */
class WheelEvent extends UIEvent_js_1.default {
    static DOM_DELTA_PIXEL = 0;
    static DOM_DELTA_LINE = 1;
    static DOM_DELTA_PAGE = 2;
    deltaX;
    deltaY;
    deltaZ;
    deltaMode;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.deltaX = eventInit?.deltaX ?? 0;
        this.deltaY = eventInit?.deltaY ?? 0;
        this.deltaZ = eventInit?.deltaZ ?? 0;
        this.deltaMode = eventInit?.deltaMode ?? 0;
    }
}
exports.default = WheelEvent;
//# sourceMappingURL=WheelEvent.cjs.map