"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("./Event.cjs"));
/**
 *
 */
class UIEvent extends Event_js_1.default {
    static NONE = 0;
    static CAPTURING_PHASE = 1;
    static AT_TARGET = 2;
    static BUBBLING_PHASE = 3;
    detail;
    layerX = 0;
    layerY = 0;
    pageX = 0;
    pageY = 0;
    view;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.detail = eventInit?.detail ?? 0;
        this.view = eventInit?.view ?? null;
    }
}
exports.default = UIEvent;
//# sourceMappingURL=UIEvent.cjs.map