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
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * DOM Point Readonly.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMPointReadOnly
 */
class DOMPointReadOnly {
    [PropertySymbol.x] = 0;
    [PropertySymbol.y] = 0;
    [PropertySymbol.z] = 0;
    [PropertySymbol.w] = 1;
    /**
     * Constructor.
     *
     * @param [x] X position.
     * @param [y] Y position.
     * @param [z] Width.
     * @param [w] Height.
     */
    constructor(x, y, z, w) {
        this[PropertySymbol.x] = x !== undefined && x !== null ? Number(x) : 0;
        this[PropertySymbol.y] = y !== undefined && y !== null ? Number(y) : 0;
        this[PropertySymbol.z] = z !== undefined && z !== null ? Number(z) : 0;
        this[PropertySymbol.w] = w !== undefined && w !== null ? Number(w) : 1;
    }
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        return this[PropertySymbol.x];
    }
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y() {
        return this[PropertySymbol.y];
    }
    /**
     * Returns z.
     *
     * @returns Z.
     */
    get z() {
        return this[PropertySymbol.z];
    }
    /**
     * Returns w.
     *
     * @returns W.
     */
    get w() {
        return this[PropertySymbol.w];
    }
    /**
     * Returns the JSON representation of the object.
     *
     * @returns JSON representation.
     */
    toJSON() {
        return {
            x: this.x,
            y: this.y,
            z: this.z,
            w: this.w
        };
    }
    /**
     * Returns a new DOMPointReadOnly object.
     *
     * @param [otherPoint] Other point.
     * @returns Cloned object.
     */
    static fromPoint(otherPoint) {
        if (!otherPoint) {
            return new this();
        }
        return new this(otherPoint.x ?? null, otherPoint.y ?? null, otherPoint.z ?? null, otherPoint.w ?? null);
    }
}
exports.default = DOMPointReadOnly;
//# sourceMappingURL=DOMPointReadOnly.cjs.map