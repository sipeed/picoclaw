"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../Event.cjs"));
/**
 *
 */
class MediaQueryListEvent extends Event_js_1.default {
    matches;
    media;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.matches = eventInit?.matches ?? false;
        this.media = eventInit?.media ?? '';
    }
}
exports.default = MediaQueryListEvent;
//# sourceMappingURL=MediaQueryListEvent.cjs.map