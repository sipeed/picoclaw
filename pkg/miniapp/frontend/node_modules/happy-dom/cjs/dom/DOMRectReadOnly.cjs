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
 * DOM Rect Readonly.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMRectReadOnly
 */
class DOMRectReadOnly {
    [PropertySymbol.x] = 0;
    [PropertySymbol.y] = 0;
    [PropertySymbol.width] = 0;
    [PropertySymbol.height] = 0;
    /**
     * Constructor.
     *
     * @param [x] X position.
     * @param [y] Y position.
     * @param [width] Width.
     * @param [height] Height.
     */
    constructor(x, y, width, height) {
        this[PropertySymbol.x] = x !== undefined && x !== null ? Number(x) : 0;
        this[PropertySymbol.y] = y !== undefined && y !== null ? Number(y) : 0;
        this[PropertySymbol.width] = width !== undefined && width !== null ? Number(width) : 0;
        this[PropertySymbol.height] = height !== undefined && height !== null ? Number(height) : 0;
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
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        return this[PropertySymbol.width];
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        return this[PropertySymbol.height];
    }
    /**
     * Returns top.
     *
     * @returns Top.
     */
    get top() {
        return Math.min(this[PropertySymbol.y], this[PropertySymbol.y] + this[PropertySymbol.height]);
    }
    /**
     * Returns right.
     *
     * @returns Right.
     */
    get right() {
        return Math.max(this[PropertySymbol.x], this[PropertySymbol.x] + this[PropertySymbol.width]);
    }
    /**
     * Returns bottom.
     *
     * @returns Bottom.
     */
    get bottom() {
        return Math.max(this[PropertySymbol.y], this[PropertySymbol.y] + this[PropertySymbol.height]);
    }
    /**
     * Returns left.
     *
     * @returns Left.
     */
    get left() {
        return Math.min(this[PropertySymbol.x], this[PropertySymbol.x] + this[PropertySymbol.width]);
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
            width: this.width,
            height: this.height,
            top: this.top,
            right: this.right,
            bottom: this.bottom,
            left: this.left
        };
    }
    /**
     * Returns a new DOMRectReadOnly object.
     *
     * @param other
     * @returns Cloned object.
     */
    static fromRect(other) {
        return new DOMRectReadOnly(other.x, other.y, other.width, other.height);
    }
}
exports.default = DOMRectReadOnly;
//# sourceMappingURL=DOMRectReadOnly.cjs.map