import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
/**
 * SVGFESpecularLightingElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFESpecularLightingElement
 */
export default class SVGFESpecularLightingElement extends SVGElement {
    // Internal properties
    [PropertySymbol.height] = null;
    [PropertySymbol.in1] = null;
    [PropertySymbol.kernelUnitLengthX] = null;
    [PropertySymbol.kernelUnitLengthY] = null;
    [PropertySymbol.result] = null;
    [PropertySymbol.specularConstant] = null;
    [PropertySymbol.specularExponent] = null;
    [PropertySymbol.surfaceScale] = null;
    [PropertySymbol.width] = null;
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
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
     * Returns kernelUnitLengthX.
     *
     * @returns KernelUnitLengthX.
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
     * Returns kernelUnitLengthY.
     *
     * @returns KernelUnitLengthY.
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
     * Returns specularConstant.
     *
     * @returns SpecularConstant.
     */
    get specularConstant() {
        if (!this[PropertySymbol.specularConstant]) {
            this[PropertySymbol.specularConstant] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('specularConstant'),
                setAttribute: (value) => this.setAttribute('specularConstant', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.specularConstant];
    }
    /**
     * Returns specularExponent.
     *
     * @returns SpecularExponent.
     */
    get specularExponent() {
        if (!this[PropertySymbol.specularExponent]) {
            this[PropertySymbol.specularExponent] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('specularExponent'),
                setAttribute: (value) => this.setAttribute('specularExponent', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.specularExponent];
    }
    /**
     * Returns surfaceScale.
     *
     * @returns SurfaceScale.
     */
    get surfaceScale() {
        if (!this[PropertySymbol.surfaceScale]) {
            this[PropertySymbol.surfaceScale] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('surfaceScale'),
                setAttribute: (value) => this.setAttribute('surfaceScale', value),
                defaultValue: 1
            });
        }
        return this[PropertySymbol.surfaceScale];
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
//# sourceMappingURL=SVGFESpecularLightingElement.js.map