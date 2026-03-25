import SVGElement from '../svg-element/SVGElement.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.cjs';
import SVGAnimatedString from '../../svg/SVGAnimatedString.cjs';
import SVGAnimatedNumber from '../../svg/SVGAnimatedNumber.cjs';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.cjs';
/**
 * SVGFEDisplacementMapElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFEDisplacementMapElement
 */
export default class SVGFEDisplacementMapElement extends SVGElement {
    static SVG_CHANNEL_UNKNOWN: number;
    static SVG_CHANNEL_R: number;
    static SVG_CHANNEL_G: number;
    static SVG_CHANNEL_B: number;
    static SVG_CHANNEL_A: number;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.in2]: SVGAnimatedString | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.scale]: SVGAnimatedNumber | null;
    [PropertySymbol.width]: SVGAnimatedLength | null;
    [PropertySymbol.x]: SVGAnimatedLength | null;
    [PropertySymbol.xChannelSelector]: SVGAnimatedEnumeration | null;
    [PropertySymbol.y]: SVGAnimatedLength | null;
    [PropertySymbol.yChannelSelector]: SVGAnimatedEnumeration | null;
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
     * Returns result.
     *
     * @returns Result.
     */
    get result(): SVGAnimatedString;
    /**
     * Returns scale.
     *
     * @returns Scale.
     */
    get scale(): SVGAnimatedNumber;
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
     * Returns x channel selector.
     *
     * @returns X channel selector.
     */
    get xChannelSelector(): SVGAnimatedEnumeration;
    /**
     * Returns y position.
     *
     * @returns Y position.
     */
    get y(): SVGAnimatedLength;
    /**
     * Returns y channel selector.
     *
     * @returns Y channel selector.
     */
    get yChannelSelector(): SVGAnimatedEnumeration;
}
//# sourceMappingURL=SVGFEDisplacementMapElement.d.ts.map