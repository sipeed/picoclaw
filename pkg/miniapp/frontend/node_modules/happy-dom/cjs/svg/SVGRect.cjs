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
 * Rect object.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGRect
 */
class SVGRect {
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute] = null;
    [PropertySymbol.setAttribute] = null;
    [PropertySymbol.attributeValue] = null;
    [PropertySymbol.readOnly] = false;
    /**
     * Constructor.
     *
     * @param illegalConstructorSymbol Illegal constructor symbol.
     * @param window Window.
     * @param [options] Options.
     * @param [options.readOnly] Read only.
     * @param [options.getAttribute] Get attribute.
     * @param [options.setAttribute] Set attribute.
     */
    constructor(illegalConstructorSymbol, window, options) {
        if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
            throw new TypeError('Illegal constructor');
        }
        this[PropertySymbol.window] = window;
        if (options) {
            this[PropertySymbol.readOnly] = !!options.readOnly;
            this[PropertySymbol.getAttribute] = options.getAttribute || null;
            this[PropertySymbol.setAttribute] = options.setAttribute || null;
        }
    }
    /**
     * Returns x value.
     *
     * @returns X value.
     */
    get x() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        if (!attributeValue) {
            return 0;
        }
        const parts = attributeValue.split(/\s+/);
        const value = Number(parts[0]);
        return isNaN(value) ? 0 : value;
    }
    /**
     * Sets x value.
     *
     * @param value X value.
     */
    set x(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'x' property on 'SVGRect': The object is read-only.`);
        }
        this[PropertySymbol.attributeValue] = `${String(typeof value === 'number' ? value : parseFloat(value))} ${this.y} ${this.width} ${this.height}`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Returns y value.
     *
     * @returns Y value.
     */
    get y() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        if (!attributeValue) {
            return 0;
        }
        const parts = attributeValue.split(/\s+/);
        const value = Number(parts[1]);
        return isNaN(value) ? 0 : value;
    }
    /**
     * Sets y value.
     *
     * @param value Y value.
     */
    set y(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'y' property on 'SVGRect': The object is read-only.`);
        }
        this[PropertySymbol.attributeValue] = `${this.x} ${String(typeof value === 'number' ? value : parseFloat(value))} ${this.width} ${this.height}`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Returns width value.
     *
     * @returns Width value.
     */
    get width() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        if (!attributeValue) {
            return 0;
        }
        const parts = attributeValue.split(/\s+/);
        const value = Number(parts[2]);
        return isNaN(value) ? 0 : value;
    }
    /**
     * Sets width value.
     *
     * @param value Width value.
     */
    set width(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'width' property on 'SVGRect': The object is read-only.`);
        }
        this[PropertySymbol.attributeValue] = `${this.x} ${this.y} ${String(typeof value === 'number' ? value : parseFloat(value))} ${this.height}`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Returns height value.
     *
     * @returns Height value.
     */
    get height() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        if (!attributeValue) {
            return 0;
        }
        const parts = attributeValue.split(/\s+/);
        const value = Number(parts[3]);
        return isNaN(value) ? 0 : value;
    }
    /**
     * Sets height value.
     *
     * @param value Height value.
     */
    set height(value) {
        if (this[PropertySymbol.readOnly]) {
            throw new this[PropertySymbol.window].TypeError(`Failed to set the 'height' property on 'SVGRect': The object is read-only.`);
        }
        this[PropertySymbol.attributeValue] = `${this.x} ${this.y} ${this.width} ${String(typeof value === 'number' ? value : parseFloat(value))}`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
}
exports.default = SVGRect;
//# sourceMappingURL=SVGRect.cjs.map