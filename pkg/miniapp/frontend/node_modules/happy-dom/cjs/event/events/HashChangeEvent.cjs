"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../Event.cjs"));
/**
 * Hash change event.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HashChangeEvent
 */
class HashChangeEvent extends Event_js_1.default {
    newURL;
    oldURL;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.newURL = eventInit?.newURL ?? '';
        this.oldURL = eventInit?.oldURL ?? '';
    }
}
exports.default = HashChangeEvent;
//# sourceMappingURL=HashChangeEvent.cjs.map