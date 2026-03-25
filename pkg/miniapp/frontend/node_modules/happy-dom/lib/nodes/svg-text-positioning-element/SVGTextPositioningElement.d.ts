import * as PropertySymbol from '../../PropertySymbol.js';
import SVGTextContentElement from '../svg-text-content-element/SVGTextContentElement.js';
import SVGAnimatedLengthList from '../../svg/SVGAnimatedLengthList.js';
import SVGAnimatedNumberList from '../../svg/SVGAnimatedNumberList.js';
/**
 * SVG Text Positioning Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGTextPositioningElement
 */
export default class SVGTextPositioningElement extends SVGTextContentElement {
    [PropertySymbol.x]: SVGAnimatedLengthList | null;
    [PropertySymbol.y]: SVGAnimatedLengthList | null;
    [PropertySymbol.dx]: SVGAnimatedLengthList | null;
    [PropertySymbol.dy]: SVGAnimatedLengthList | null;
    [PropertySymbol.rotate]: SVGAnimatedNumberList | null;
    /**
     * Returns x.
     *
     * @returns X.
     */
    get x(): SVGAnimatedLengthList;
    /**
     * Returns y.
     *
     * @returns Y.
     */
    get y(): SVGAnimatedLengthList;
    /**
     * Returns dx.
     *
     * @returns DX.
     */
    get dx(): SVGAnimatedLengthList;
    /**
     * Returns dy.
     *
     * @returns DY.
     */
    get dy(): SVGAnimatedLengthList;
    /**
     * Returns rotate.
     *
     * @returns Rotate.
     */
    get rotate(): SVGAnimatedNumberList;
}
//# sourceMappingURL=SVGTextPositioningElement.d.ts.map