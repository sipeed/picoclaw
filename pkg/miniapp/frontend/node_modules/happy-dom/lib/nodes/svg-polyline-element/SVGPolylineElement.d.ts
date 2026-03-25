import SVGGeometryElement from '../svg-geometry-element/SVGGeometryElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import SVGPointList from '../../svg/SVGPointList.js';
/**
 * SVG Polyline Element.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/SVGPolylineElement
 */
export default class SVGPolylineElement extends SVGGeometryElement {
    [PropertySymbol.animatedPoints]: SVGPointList | null;
    [PropertySymbol.points]: SVGPointList | null;
    /**
     * Returns animated points.
     *
     * @returns Animated points.
     */
    get animatedPoints(): SVGPointList;
    /**
     * Returns points.
     *
     * @returns Points.
     */
    get points(): SVGPointList;
}
//# sourceMappingURL=SVGPolylineElement.d.ts.map