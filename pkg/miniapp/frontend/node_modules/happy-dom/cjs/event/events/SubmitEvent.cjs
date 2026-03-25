"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../Event.cjs"));
/**
 * An event triggered by form submit buttons.
 */
class SubmitEvent extends Event_js_1.default {
    submitter;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.submitter = eventInit?.submitter ?? null;
    }
}
exports.default = SubmitEvent;
//# sourceMappingURL=SubmitEvent.cjs.map