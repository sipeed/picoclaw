"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../Event.cjs"));
/**
 *
 */
class ProgressEvent extends Event_js_1.default {
    lengthComputable;
    loaded;
    total;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type);
        this.lengthComputable = eventInit?.lengthComputable ?? false;
        this.loaded = eventInit?.loaded ?? 0;
        this.total = eventInit?.total ?? 0;
    }
}
exports.default = ProgressEvent;
//# sourceMappingURL=ProgressEvent.cjs.map