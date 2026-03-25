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
const DOMRectReadOnly_js_1 = __importDefault(require("./DOMRectReadOnly.cjs"));
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
/**
 * DOM Rect.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMRect
 */
class DOMRect extends DOMRectReadOnly_js_1.default {
    /**
     * Sets x.
     *
     * @param value X.
     */
    set x(value) {
        this[PropertySymbol.x] = value;
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
     * Sets y.
     *
     * @param value Y.
     */
    set y(value) {
        this[PropertySymbol.y] = value;
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
     * Sets width.
     *
     * @param value Width.
     */
    set width(value) {
        this[PropertySymbol.width] = value;
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
     * Sets height.
     *
     * @param value Height.
     */
    set height(value) {
        this[PropertySymbol.height] = value;
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
     * Returns a new DOMRect object.
     *
     * @param other
     * @returns Cloned object.
     */
    static fromRect(other) {
        return new DOMRect(other.x, other.y, other.width, other.height);
    }
}
exports.default = DOMRect;
//# sourceMappingURL=DOMRect.cjs.map