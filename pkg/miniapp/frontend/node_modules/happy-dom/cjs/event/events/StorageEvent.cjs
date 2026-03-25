"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const Event_js_1 = __importDefault(require("../Event.cjs"));
/**
 *
 */
class StorageEvent extends Event_js_1.default {
    key;
    oldValue;
    newValue;
    url;
    storageArea;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type);
        this.key = eventInit?.key ?? null;
        this.oldValue = eventInit?.oldValue ?? null;
        this.newValue = eventInit?.newValue ?? null;
        this.url = eventInit?.url ?? '';
        this.storageArea = eventInit?.storageArea ?? null;
    }
}
exports.default = StorageEvent;
//# sourceMappingURL=StorageEvent.cjs.map