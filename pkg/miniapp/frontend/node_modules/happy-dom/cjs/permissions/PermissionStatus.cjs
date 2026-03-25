"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const EventTarget_js_1 = __importDefault(require("../event/EventTarget.cjs"));
/**
 * Permission status.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/PermissionStatus
 */
class PermissionStatus extends EventTarget_js_1.default {
    state;
    onchange = null;
    /**
     * Constructor.
     *
     * @param [state] State.
     */
    constructor(state = 'granted') {
        super();
        this.state = state;
    }
}
exports.default = PermissionStatus;
//# sourceMappingURL=PermissionStatus.cjs.map