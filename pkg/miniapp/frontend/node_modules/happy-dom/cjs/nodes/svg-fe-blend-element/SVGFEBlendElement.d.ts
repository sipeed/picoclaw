import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
/**
 * SVG FE Blend Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEBlendElement
 */
export default class SVGFEBlendElement extends SVGElement {
    static SVG_FEBLEND_MODE_UNKNOWN: number;
    static SVG_FEBLEND_MODE_NORMAL: number;
    static SVG_FEBLEND_MODE_MULTIPLY: number;
    static SVG_FEBLEND_MODE_SCREEN: number;
    static SVG_FEBLEND_MODE_DARKEN: number;
    static SVG_FEBLEND_MODE_LIGHTEN: number;
    static SVG_FEBLEND_MODE_OVERLAY: number;
    static SVG_FEBLEND_MODE_COLOR_DODGE: number;
    static SVG_FEBLEND_MODE_COLOR_BURN: number;
    static SVG_FEBLEND_MODE_HARD_LIGHT: number;
    static SVG_FEBLEND_MODE_SOFT_LIGHT: number;
    static SVG_FEBLEND_MODE_DIFFERENCE: number;
    static SVG_FEBLEND_MODE_EXCLUSION: number;
    static SVG_FEBLEND_MODE_HUE: number;
    static SVG_FEBLEND_MODE_SATURATION: number;
    static SVG_FEBLEND_MODE_COLOR: number;
    static SVG_FEBLEND_MODE_LUMINOSITY: number;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.in2]: SVGAnimatedString | null;
    [PropertySymbol.mode]: SVGAnimatedEnumeration | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    /**
     * Returns height.
     *
     * @returns Height.
     */
    get height(): SVGAnimatedLength;
    /**
     * Returns in1.
     *
     * @returns In1.
     */
    get in1(): SVGAnimatedString;
    /**
     * Returns in2.
     *
     * @returns In2.
     */
    get in2(): SVGAnimatedString;
    /**
     * Returns mode.
     *
     * @returns Mode.
     */
    get mode(): SVGAnimatedEnumeration;
    /**
     * Returns width.
     *
     * @returns Width.
     */
    get width(): SVGAnimatedLength;
    /**
     * Returns x position.
     *
     * @returns X position.
     */
    get x(): SVGAnimatedLength;
    /**
     * Returns y position.
     *
     * @returns Y position.
     */
    get y(): SVGAnimatedLength;
}
//# sourceMappingURL=SVGFEBlendElement.d.ts.map