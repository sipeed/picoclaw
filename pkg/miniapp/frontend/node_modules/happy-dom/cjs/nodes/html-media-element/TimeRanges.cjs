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
Object.defineProperty(exports, "__esModule", { value: true });
const PropertySymbol = __importStar(require("../../PropertySymbol.cjs"));
/**
 * When loading a media resource for use by an <audio> or <video> element, the TimeRanges interface is used for representing the time ranges of the media resource that have been buffered, the time ranges that have been played, and the time ranges that are seekable.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TimeRanges
 */
class TimeRanges {
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     */
    constructor(illegalConstructorSymbol) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
    }
    /**
     * Returns length.
     */
    get length() {
        return 0;
    }
    /**
     * Returns `Symbol.toStringTag`.
     *
     * @returns `Symbol.toStringTag`.
     */
    get [Symbol.toStringTag]() {
        return 'TimeRanges';
    }
    /**
     * Returns `[object NodeList]`.
     *
     * @returns `[object NodeList]`.
     */
    toLocaleString() {
        return '[object TimeRanges]';
    }
    /**
     * Returns `[object NodeList]`.
     *
     * @returns `[object NodeList]`.
     */
    toString() {
        return '[object TimeRanges]';
    }
    /**
     * Returns start.
     *
     * @param _index Index.
     * @returns Start.
     */
    start(_index) {
        return 0;
    }
    /**
     * Returns end.
     *
     * @param _index Index.
     * @returns End.
     */
    end(_index) {
        return 0;
    }
}
exports.default = TimeRanges;
//# sourceMappingURL=TimeRanges.cjs.map