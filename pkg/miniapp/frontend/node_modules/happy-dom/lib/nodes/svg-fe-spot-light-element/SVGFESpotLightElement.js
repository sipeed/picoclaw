import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.js';
/**
 * SVGFESpotLightElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFESpotLightElement
 */
export default class SVGFESpotLightElement extends SVGElement {
    // Internal properties
    [PropertySymbol.x] = null;
    [PropertySymbol.y] = null;
    [PropertySymbol.z] = null;
    [PropertySymbol.pointsAtX] = null;
    [PropertySymbol.pointsAtY] = null;
    [PropertySymbol.pointsAtZ] = null;
    [PropertySymbol.specularExponent] = null;
    [PropertySymbol.limitingConeAngle] = null;
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x() {
        if (!this[PropertySymbol.x]) {
            this[PropertySymbol.x] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('x'),
                setAttribute: (value) => this.setAttribute('x', value)
            });
        }
        return this[PropertySymbol.x];
    }
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y() {
        if (!this[PropertySymbol.y]) {
            this[PropertySymbol.y] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('y'),
                setAttribute: (value) => this.setAttribute('y', value)
            });
        }
        return this[PropertySymbol.y];
    }
    /**
     * Returns z.
     *
     * @returns Z.
     */
    get z() {
        if (!this[PropertySymbol.z]) {
            this[PropertySymbol.z] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('z'),
                setAttribute: (value) => this.setAttribute('z', value)
            });
        }
        return this[PropertySymbol.z];
    }
    /**
     * Returns pointsAtX.
     *
     * @returns PointsAtX.
     */
    get pointsAtX() {
        if (!this[PropertySymbol.pointsAtX]) {
            this[PropertySymbol.pointsAtX] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('pointsAtX'),
                setAttribute: (value) => this.setAttribute('pointsAtX', value)
            });
        }
        return this[PropertySymbol.pointsAtX];
    }
    /**
     * Returns pointsAtY.
     *
     * @returns PointsAtY.
     */
    get pointsAtY() {
        if (!this[PropertySymbol.pointsAtY]) {
            this[PropertySymbol.pointsAtY] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('pointsAtY'),
                setAttribute: (value) => this.setAttribute('pointsAtY', value)
            });
        }
        return this[PropertySymbol.pointsAtY];
    }
    /**
     * Returns pointsAtZ.
     *
     * @returns PointsAtZ.
     */
    get pointsAtZ() {
        if (!this[PropertySymbol.pointsAtZ]) {
            this[PropertySymbol.pointsAtZ] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('pointsAtZ'),
                setAttribute: (value) => this.setAttribute('pointsAtZ', value)
            });
        }
        return this[PropertySymbol.pointsAtZ];
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
     * Returns limitingConeAngle.
     *
     * @returns LimitingConeAngle.
     */
    get limitingConeAngle() {
        if (!this[PropertySymbol.limitingConeAngle]) {
            this[PropertySymbol.limitingConeAngle] = new SVGAnimatedNumber(PropertySymbol.illegalConstructor, this[PropertySymbol.window], {
                getAttribute: () => this.getAttribute('limitingConeAngle'),
                setAttribute: (value) => this.setAttribute('limitingConeAngle', value)
            });
        }
        return this[PropertySymbol.limitingConeAngle];
    }
}
//# sourceMappingURL=SVGFESpotLightElement.js.map