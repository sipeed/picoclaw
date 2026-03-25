"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../Event.cjs"));
/**
 *
 */
class AnimationEvent extends Event_js_1.default {
    animationName;
    elapsedTime;
    pseudoElement;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.animationName = eventInit?.animationName ?? '';
        this.elapsedTime = eventInit?.elapsedTime ?? 0;
        this.pseudoElement = eventInit?.pseudoElement ?? '';
    }
}
exports.default = AnimationEvent;
//# sourceMappingURL=AnimationEvent.cjs.map