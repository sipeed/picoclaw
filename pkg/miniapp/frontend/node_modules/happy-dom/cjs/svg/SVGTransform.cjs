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
const PropertySymbol = __importStar(require("../PropertySymbol.cjs"));
const SVGMatrix_js_1 = __importDefault(require("./SVGMatrix.cjs"));
const SVGTransformTypeEnum_js_1 = __importDefault(require("./SVGTransformTypeEnum.cjs"));
const TRANSFORM_REGEXP = /([a-zA-Z0-9]+)\(([^)]+)\)/;
const TRANSFORM_PARAMETER_SPLIT_REGEXP = /[\s,]+/;
/**
 * SVG transform.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTransform
 */
class SVGTransform {
    static SVG_TRANSFORM_UNKNOWN = SVGTransformTypeEnum_js_1.default.unknown;
    static SVG_TRANSFORM_MATRIX = SVGTransformTypeEnum_js_1.default.matrix;
    static SVG_TRANSFORM_TRANSLATE = SVGTransformTypeEnum_js_1.default.translate;
    static SVG_TRANSFORM_SCALE = SVGTransformTypeEnum_js_1.default.scale;
    static SVG_TRANSFORM_ROTATE = SVGTransformTypeEnum_js_1.default.rotate;
    static SVG_TRANSFORM_SKEWX = SVGTransformTypeEnum_js_1.default.skewX;
    static SVG_TRANSFORM_SKEWY = SVGTransformTypeEnum_js_1.default.skewY;
    // Internal properties
    [PropertySymbol.window];
    [PropertySymbol.getAttribute];
    [PropertySymbol.setAttribute];
    [PropertySymbol.attributeValue] = null;
    [PropertySymbol.readOnly] = false;
    [PropertySymbol.matrix] = null;
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
     * Returns type.
     *
     * @returns Type.
     */
    get type() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        const match = attributeValue?.match(TRANSFORM_REGEXP);
        if (!match) {
            return SVGTransformTypeEnum_js_1.default.unknown;
        }
        switch (match[1]) {
            case 'matrix':
                return SVGTransformTypeEnum_js_1.default.matrix;
            case 'translate':
                return SVGTransformTypeEnum_js_1.default.translate;
            case 'rotate':
                return SVGTransformTypeEnum_js_1.default.rotate;
            case 'scale':
                return SVGTransformTypeEnum_js_1.default.scale;
            case 'skewX':
                return SVGTransformTypeEnum_js_1.default.skewX;
            case 'skewY':
                return SVGTransformTypeEnum_js_1.default.skewY;
        }
        return 0;
    }
    /**
     * Returns angle.
     *
     * @returns Angle.
     */
    get angle() {
        const attributeValue = this[PropertySymbol.getAttribute]
            ? this[PropertySymbol.getAttribute]()
            : this[PropertySymbol.attributeValue];
        const match = attributeValue?.match(TRANSFORM_REGEXP);
        if (!match) {
            return 0;
        }
        const angle = parseFloat(match[2].trim().split(TRANSFORM_PARAMETER_SPLIT_REGEXP)[0]);
        if (isNaN(angle)) {
            return 0;
        }
        switch (match[1]) {
            case 'rotate':
            case 'skewX':
            case 'skewY':
                return angle;
        }
        return 0;
    }
    /**
     * Returns matrix.
     *
     * @returns Matrix.
     */
    get matrix() {
        if (!this[PropertySymbol.matrix]) {
            this[PropertySymbol.matrix] = new SVGMatrix_js_1.default(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                readOnly: this[PropertySymbol.readOnly],
                getAttribute: () => {
                    if (this[PropertySymbol.getAttribute]) {
                        return this[PropertySymbol.getAttribute]();
                    }
                    return this[PropertySymbol.attributeValue];
                },
                setAttribute: (value) => {
                    this[PropertySymbol.attributeValue] = value;
                    if (this[PropertySymbol.setAttribute]) {
                        this[PropertySymbol.setAttribute](value);
                        return;
                    }
                }
            });
        }
        return this[PropertySymbol.matrix];
    }
    /**
     * Set matrix.
     *
     * @param matrix Matrix.
     */
    setMatrix(matrix) {
        if (!(matrix instanceof SVGMatrix_js_1.default)) {
            throw new TypeError('Failed to set the "matrix" property on "SVGTransform": The provided value is not of type "SVGMatrix".');
        }
        if (this[PropertySymbol.readOnly]) {
            return;
        }
        if (this[PropertySymbol.matrix]) {
            this[PropertySymbol.matrix][PropertySymbol.getAttribute] = null;
            this[PropertySymbol.matrix][PropertySymbol.setAttribute] = null;
        }
        matrix[PropertySymbol.getAttribute] = () => {
            if (this[PropertySymbol.getAttribute]) {
                return this[PropertySymbol.getAttribute]();
            }
            return this[PropertySymbol.attributeValue];
        };
        matrix[PropertySymbol.setAttribute] = (value) => {
            this[PropertySymbol.attributeValue] = value;
            if (this[PropertySymbol.setAttribute]) {
                this[PropertySymbol.setAttribute](value);
                return;
            }
        };
        this[PropertySymbol.matrix] = matrix;
        if (matrix[PropertySymbol.attributeValue] !== this[PropertySymbol.attributeValue]) {
            this[PropertySymbol.attributeValue] = matrix[PropertySymbol.attributeValue];
            if (this[PropertySymbol.setAttribute]) {
                this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
            }
        }
    }
    /**
     * Set translate.
     *
     * @param x X.
     * @param y Y.
     */
    setTranslate(x, y) {
        if (arguments.length < 2) {
            throw new TypeError(`Failed to execute 'setTranslate' on 'SVGTransform': 2 arguments required, but only ${arguments.length} present.`);
        }
        x = Number(x);
        y = Number(y);
        if (isNaN(x) || isNaN(y)) {
            throw new TypeError(`Failed to execute 'setTranslate' on 'SVGTransform':  The provided float value is non-finite.`);
        }
        if (this[PropertySymbol.readOnly]) {
            return;
        }
        this[PropertySymbol.attributeValue] = `translate(${x} ${y})`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Set scale.
     *
     * @param x X.
     * @param y Y.
     */
    setScale(x, y) {
        if (arguments.length < 2) {
            throw new TypeError(`Failed to execute 'setScale' on 'SVGTransform': 2 arguments required, but only ${arguments.length} present.`);
        }
        x = Number(x);
        y = Number(y);
        if (isNaN(x) || isNaN(y)) {
            throw new TypeError(`Failed to execute 'setScale' on 'SVGTransform':  The provided float value is non-finite.`);
        }
        if (this[PropertySymbol.readOnly]) {
            return;
        }
        this[PropertySymbol.attributeValue] = `scale(${x} ${y})`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Set rotate.
     *
     * @param angle Angle.
     * @param x X.
     * @param y Y.
     */
    setRotate(angle, x, y) {
        if (arguments.length < 3) {
            throw new TypeError(`Failed to execute 'setRotate' on 'SVGTransform': 3 arguments required, but only ${arguments.length} present.`);
        }
        angle = Number(angle);
        x = Number(x);
        y = Number(y);
        if (isNaN(angle) || isNaN(x) || isNaN(y)) {
            throw new TypeError(`Failed to execute 'setRotate' on 'SVGTransform':  The provided float value is non-finite.`);
        }
        if (this[PropertySymbol.readOnly]) {
            return;
        }
        this[PropertySymbol.attributeValue] = `rotate(${angle} ${x} ${y})`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Set skew x.
     *
     * @param angle Angle.
     */
    setSkewX(angle) {
        if (arguments.length < 1) {
            throw new TypeError(`Failed to execute 'setSkewX' on 'SVGTransform': 1 arguments required, but only ${arguments.length} present.`);
        }
        angle = Number(angle);
        if (isNaN(angle)) {
            throw new TypeError(`Failed to execute 'setSkewX' on 'SVGTransform':  The provided float value is non-finite.`);
        }
        if (this[PropertySymbol.readOnly]) {
            return;
        }
        this[PropertySymbol.attributeValue] = `skewX(${angle})`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
    /**
     * Set skew y.
     *
     * @param angle Angle.
     */
    setSkewY(angle) {
        if (arguments.length < 1) {
            throw new TypeError(`Failed to execute 'setSkewY' on 'SVGTransform': 1 arguments required, but only ${arguments.length} present.`);
        }
        angle = Number(angle);
        if (isNaN(angle)) {
            throw new TypeError(`Failed to execute 'setSkewY' on 'SVGTransform':  The provided float value is non-finite.`);
        }
        if (this[PropertySymbol.readOnly]) {
            return;
        }
        this[PropertySymbol.attributeValue] = `skewY(${angle})`;
        if (this[PropertySymbol.setAttribute]) {
            this[PropertySymbol.setAttribute](this[PropertySymbol.attributeValue]);
        }
    }
}
exports.default = SVGTransform;
//# sourceMappingURL=SVGTransform.cjs.map