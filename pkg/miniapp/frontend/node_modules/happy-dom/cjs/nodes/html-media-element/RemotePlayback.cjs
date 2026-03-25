"use strict";
var __createBinding = (this && this.__createBinding) || (Object.create ? (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    var desc = Object.getOwnPropertyDescriptor(m, k);
    if (!desc || ("get" in desc ? !m.__esModule : desc.writable || desc.configurable)) {
      desc = { enumerable: true, get: function() { return m[k]; } };
    }
    Object.defineProperty(o, k2, desc);
}) : (function(o, m, k, k2) {
    if (k2 === undefined) k2 = k;
    o[k2] = m[k];
}));
var __setModuleDefault = (this && this.__setModuleDefault) || (Object.create ? (function(o, v) {
    Object.defineProperty(o, "default", { enumerable: true, value: v });
}) : function(o, v) {
    o["default"] = v;
});
var __importStar = (this && this.__importStar) || function (mod) {
    if (mod && mod.__esModule) return mod;
    var result = {};
    if (mod != null) for (var k in mod) if (k !== "default" && Object.prototype.hasOwnProperty.call(mod, k)) __createBinding(result, mod, k);
    __setModuleDefault(result, mod);
    return result;
};
var __importDefault = (this && this.__importDefault) || function (mod) {
    return (mod && mod.__esModule) ? mod : { "default": mod };
};
Object.defineProperty(exports, "__esModule", { value: true });
const EventTarget_js_1 = __importDefault(require("../../event/EventTarget.cjs"));
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * RemotePlayback.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/RemotePlayback
 */
class RemotePlayback extends EventTarget_js_1.default {
    // Internal properties
    [PropertySymbol.state] = 'disconnected';
    // Events
    onconnecting = null;
    onconnect = null;
    ondisconnect = null;
    /**
     * Returns the state of the remote playback.
     */
    get state() {
        return this[PropertySymbol.state];
    }
    /**
     * Watches the list of available remote playback devices and returns a Promise that resolves with a callbackId of an available remote playback device.
     *
     * @returns Promise.
     */
    async watchAvailability() {
        // TODO: Implement
    }
    /**
     * Cancels the request to monitor the availability of remote playback devices.
     */
    cancelWatchAvailability() {
        // TODO: Implement
    }
    /**
     * Prompts the user to select and give permission to connect to a remote playback device.
     */
    prompt() {
        // TODO: Implement
    }
}
exports.default = RemotePlayback;
//# sourceMappingURL=RemotePlayback.cjs.map