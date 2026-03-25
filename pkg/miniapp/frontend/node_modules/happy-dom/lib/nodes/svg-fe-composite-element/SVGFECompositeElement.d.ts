import SVGElement from '../svg-element/SVGElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGAnimatedLength from '../../svg/SVGAnimatedLength.js';
import SVGAnimatedEnumeration from '../../svg/SVGAnimatedEnumeration.js';
import SVGAnimatedString from '../../svg/SVGAnimatedString.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';
/**
 * SVGFECompositeElement.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGFECompositeElement
 */
export default class SVGFECompositeElement extends SVGElement {
    static SVG_FECOMPOSITE_OPERATOR_UNKNOWN: number;
    static SVG_FECOMPOSITE_OPERATOR_OVER: number;
    static SVG_FECOMPOSITE_OPERATOR_IN: number;
    static SVG_FECOMPOSITE_OPERATOR_OUT: number;
    static SVG_FECOMPOSITE_OPERATOR_ATOP: number;
    static SVG_FECOMPOSITE_OPERATOR_XOR: number;
    static SVG_FECOMPOSITE_OPERATOR_ARITHMETIC: number;
    [PropertySymbol.height]: SVGAnimatedLength | null;
    [PropertySymbol.in1]: SVGAnimatedString | null;
    [PropertySymbol.result]: SVGAnimatedString | null;
    [PropertySymbol.type]: SVGAnimatedEnumeration | null;
    [PropertySymbol.values]: SVGAnimatedNumberList | null;
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
     * Returns result.
     *
     * @returns Result.
     */
    get result(): SVGAnimatedString;
    /**
     * Returns type.
     *
     * @returns Type.
     */
    get type(): SVGAnimatedEnumeration;
    /**
     * Returns values.
     *
     * @returns Values.
     */
    get values(): SVGAnimatedNumberList;
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
//# sourceMappingURL=SVGFECompositeElement.d.ts.map