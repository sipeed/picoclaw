"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../Event.cjs"));
/**
 * Media Stream Track Event.
 */
class MediaStreamTrackEvent extends Event_js_1.default {
    track;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.track = eventInit?.track ?? null;
    }
}
exports.default = MediaStreamTrackEvent;
//# sourceMappingURL=MediaStreamTrackEvent.cjs.map