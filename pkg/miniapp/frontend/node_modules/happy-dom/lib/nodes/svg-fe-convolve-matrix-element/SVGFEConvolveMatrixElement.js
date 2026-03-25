import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedBoolean from '../../svg/SVGAnimatedBoolean.js';
import SVGAnimatedInteger from '../../svg/SVGAnimatedInteger.js';
/**
 * SVGFEConvolveMatrixElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEConvolveMatrixElement
 */
export default class SVGFEConvolveMatrixElement extends SVGElement {
    // Static properties
    static SVG_EDGEMODE_UNKNOWN = 0;
    static SVG_EDGEMODE_DUPLICATE = 1;
    static SVG_EDGEMODE_WRAP = 2;
    static SVG_EDGEMODE_NONE = 3;
    // Internal properties
    [PropertySymbol.bias] = null;
    [PropertySymbol.divisor] = null;
    [PropertySymbol.edgeMode] = null;
    [PropertySymbol.height] = null;
    [PropertySymbol.in1] = null;
    [PropertySymbol.kernelMatrix] = null;
    [PropertySymbol.kernelUnitLengthX] = null;
    [PropertySymbol.kernelUnitLengthY] = null;
    [PropertySymbol.orderX] = null;
    [PropertySymbol.orderY] = null;
    [PropertySymbol.preserveAlpha] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.targetX] = null;
    [PropertySymbol.targetY] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    /**
     * Returns bias.
     *
     * @returns Bias.
     */
    get bias() {
        if (!this[PropertySymbol.bias]) {
            this[PropertySymbol.bias] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('bias'),
                setAttribute: (value) => this.setAttribute('bias', value)
            });
        }
        return this[PropertySymbol.bias];
    }
    /**
     * Returns divisor.
     *
     * @returns Divisor.
     */
    get divisor() {
        if (!this[PropertySymbol.divisor]) {
            this[PropertySymbol.divisor] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('divisor'),
                setAttribute: (value) => this.setAttribute('divisor', value)
            });
        }
        return this[PropertySymbol.divisor];
    }
    /**
     * Returns edge mode.
     *
     * @returns Edge mode.
     */
    get edgeMode() {
        if (!this[PropertySymbol.edgeMode]) {
            this[PropertySymbol.edgeMode] = new SVGAnimatedEnumeration(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('edgeMode'),
                setAttribute: (value) => this.setAttribute('edgeMode', value),
                values: ['duplicate', 'wrap', 'none'],
                defaultValue: 'duplicate'
            });
        }
        return this[PropertySymbol.edgeMode];
    }
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height() {
        if (!this[PropertySymbol.height]) {
            this[PropertySymbol.height] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('height'),
                setAttribute: (value) => this.setAttribute('height', value)
            });
        }
        return this[PropertySymbol.height];
    }
    /**
     * Returns in1.
     *
     * @returns In1.
     */
    get in1() {
        if (!this[PropertySymbol.in1]) {
            this[PropertySymbol.in1] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('in'),
                setAttribute: (value) => this.setAttribute('in', value)
            });
        }
        return this[PropertySymbol.in1];
    }
    /**
     * Returns kernel matrix.
     *
     * @returns Kernel matrix.
     */
    get kernelMatrix() {
        if (!this[PropertySymbol.kernelMatrix]) {
            this[PropertySymbol.kernelMatrix] = new SVGAnimatedNumberList(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('kernelMatrix'),
                setAttribute: (value) => this.setAttribute('kernelMatrix', value)
            });
        }
        return this[PropertySymbol.kernelMatrix];
    }
    /**
     * Returns kernel unit length x.
     *
     * @returns Kernel unit length x.
     */
    get kernelUnitLengthX() {
        if (!this[PropertySymbol.kernelUnitLengthX]) {
            this[PropertySymbol.kernelUnitLengthX] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('kernelUnitLengthX'),
                setAttribute: (value) => this.setAttribute('kernelUnitLengthX', value)
            });
        }
        return this[PropertySymbol.kernelUnitLengthX];
    }
    /**
     * Returns kernel unit length y.
     *
     * @returns Kernel unit length y.
     */
    get kernelUnitLengthY() {
        if (!this[PropertySymbol.kernelUnitLengthY]) {
            this[PropertySymbol.kernelUnitLengthY] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('kernelUnitLengthY'),
                setAttribute: (value) => this.setAttribute('kernelUnitLengthY', value)
            });
        }
        return this[PropertySymbol.kernelUnitLengthY];
    }
    /**
     * Returns order x.
     *
     * @returns Order x.
     */
    get orderX() {
        if (!this[PropertySymbol.orderX]) {
            this[PropertySymbol.orderX] = new SVGAnimatedInteger(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('orderX'),
                setAttribute: (value) => this.setAttribute('orderX', value)
            });
        }
        return this[PropertySymbol.orderX];
    }
    /**
     * Returns order y.
     *
     * @returns Order y.
     */
    get orderY() {
        if (!this[PropertySymbol.orderY]) {
            this[PropertySymbol.orderY] = new SVGAnimatedInteger(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('orderY'),
                setAttribute: (value) => this.setAttribute('orderY', value)
            });
        }
        return this[PropertySymbol.orderY];
    }
    /**
     * Returns preserve alpha.
     *
     * @returns Preserve alpha.
     */
    get preserveAlpha() {
        if (!this[PropertySymbol.preserveAlpha]) {
            this[PropertySymbol.preserveAlpha] = new SVGAnimatedBoolean(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('preserveAlpha'),
                setAttribute: (value) => this.setAttribute('preserveAlpha', value)
            });
        }
        return this[PropertySymbol.preserveAlpha];
    }
    /**
     * Returns result.
     *
     * @returns Result.
     */
    get result() {
        if (!this[PropertySymbol.result]) {
            this[PropertySymbol.result] = new SVGAnimatedString(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('result'),
                setAttribute: (value) => this.setAttribute('result', value)
            });
        }
        return this[PropertySymbol.result];
    }
    /**
     * Returns target x.
     *
     * @returns Target x.
     */
    get targetX() {
        if (!this[PropertySymbol.targetX]) {
            this[PropertySymbol.targetX] = new SVGAnimatedInteger(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('targetX'),
                setAttribute: (value) => this.setAttribute('targetX', value)
            });
        }
        return this[PropertySymbol.targetX];
    }
    /**
     * Returns target y.
     *
     * @returns Target y.
     */
    get targetY() {
        if (!this[PropertySymbol.targetY]) {
            this[PropertySymbol.targetY] = new SVGAnimatedInteger(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('targetY'),
                setAttribute: (value) => this.setAttribute('targetY', value)
            });
        }
        return this[PropertySymbol.targetY];
    }
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width() {
        if (!this[PropertySymbol.width]) {
            this[PropertySymbol.width] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('width'),
                setAttribute: (value) => this.setAttribute('width', value)
            });
        }
        return this[PropertySymbol.width];
    }
    /**
     * Returns x position.
     *
     * @returns X position.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x'),
                setAttribute: (value) => this.setAttribute('x', value)
            });
        }
        return this[PropertySymbol.x];
    }
    /**
     * Returns y position.
     *
     * @returns Y position.
     */
    get y() {
        if (!this[PropertySymbol.y]) {
            this[PropertySymbol.y] = new SVGAnimatedLength(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
}
//# sourceMappingURL=SVGFEConvolveMatrixElement.js.map