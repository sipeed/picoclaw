"use strict";
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const EventTarget_js_1 = __importDefault(require("./EventTarget.cjs"));
/**
 * Message port.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MessagePort
 */
class MessagePort extends EventTarget_js_1.default {
    /**
     * Sends a message from the port, and optionally, transfers ownership of objects to other browsing contexts.
     *
     * @param _message Message.
     * @param _transerList Transfer list.
     */
    postMessage(_message, _transerList) {
        // TODO: Implement
    }
    /**
     * Starts the sending of messages queued on the port.
     */
    start() {
        // TODO: Implement
    }
    /**
     * Disconnects the port, so it is no longer active. This stops the flow of messages to that port.
     */
    close() {
        // TODO: Implement
    }
}
exports.default = MessagePort;
//# sourceMappingURL=MessagePort.cjs.map