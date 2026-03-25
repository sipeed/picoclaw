"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../Event.cjs"));
/**
 * Message event.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MessageEvent
 */
class MessageEvent extends Event_js_1.default {
    data;
    origin;
    lastEventId;
    source;
    ports;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.data = eventInit?.data ?? null;
        this.origin = eventInit?.origin ?? '';
        this.lastEventId = eventInit?.lastEventId ?? '';
        this.source = eventInit?.source ?? null;
        this.ports = eventInit?.ports ?? [];
    }
}
exports.default = MessageEvent;
//# sourceMappingURL=MessageEvent.cjs.map