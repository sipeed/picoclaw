import * as PropertySymbol from '../PropertySymbol.js';
import SVGMatrix from './SVGMatrix.js';
import SVGTransformTypeEnum from './SVGTransformTypeEnum.js';
const TRANSFORM_REGEXP = /([a-zA-Z0-9]+)\(([^)]+)\)/;
const TRANSFORM_PARAMETER_SPLIT_REGEXP = /[\s,]+/;
/**
 * SVG transform.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTransform
 */
export default class SVGTransform {
    static SVG_TRANSFORM_UNKNOWN = SVGTransformTypeEnum.unknown;
    static SVG_TRANSFORM_MATRIX = SVGTransformTypeEnum.matrix;
    static SVG_TRANSFORM_TRANSLATE = SVGTransformTypeEnum.translate;
    static SVG_TRANSFORM_SCALE = SVGTransformTypeEnum.scale;
    static SVG_TRANSFORM_ROTATE = SVGTransformTypeEnum.rotate;
    static SVG_TRANSFORM_SKEWX = SVGTransformTypeEnum.skewX;
    static SVG_TRANSFORM_SKEWY = SVGTransformTypeEnum.skewY;
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
            return SVGTransformTypeEnum.unknown;
        }
        switch (match[1]) {
            case 'matrix':
                return SVGTransformTypeEnum.matrix;
            case 'translate':
                return SVGTransformTypeEnum.translate;
            case 'rotate':
                return SVGTransformTypeEnum.rotate;
            case 'scale':
                return SVGTransformTypeEnum.scale;
            case 'skewX':
                return SVGTransformTypeEnum.skewX;
            case 'skewY':
                return SVGTransformTypeEnum.skewY;
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
            this[PropertySymbol.matrix] = new SVGMatrix(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
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
        if (!(matrix instanceof SVGMatrix)) {
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
//# sourceMappingURL=SVGTransform.js.map