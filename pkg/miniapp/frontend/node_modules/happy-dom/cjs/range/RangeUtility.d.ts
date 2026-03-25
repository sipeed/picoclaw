import Node from '../nodes/node/Node.cjs';
import Range from './Range.cjs';
import IRangeBoundaryPoint from './IRangeBoundaryPoint.cjs';
/**
 * Range utility.
 *
 * Based on:
 * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/range/boundary-point.js.
 */
export default class RangeUtility {
    /**
     * Compares boundary points.
     *
     * Based on logic from:
     * https://github.com/jsdom/jsdom/blob/master/lib/jsdom/living/range/boundary-point.js
     *
     * @see https://dom.spec.whatwg.org/#concept-range-bp-after
     * @param pointA Point A.
     * @param pointB Point B.
     * @returns A number, -1, 0, or 1, indicating whether the corresponding boundary-point of the Range is respectively before, equal to, or after the corresponding boundary-point of sourceRange.
     */
    static compareBoundaryPointsPosition(pointA: IRangeBoundaryPoint, pointB: IRangeBoundaryPoint): number;
    /**
     * Validates a boundary point.
     *
     * @throws DOMException
     * @param point Boundary point.
     */
    static validateBoundaryPoint(point: IRangeBoundaryPoint): void;
    /**
     * Returns "true" if contained.
     *
     * @param node Node.
     * @param range Range.
     * @returns "true" if contained.
     */
    static isContained(node: Node, range: Range): boolean;
    /**
     * Returns "true" if partially contained.
     *
     * @param node Node.
     * @param range Range.
     * @returns "true" if partially contained.
     */
    static isPartiallyContained(node: Node, range: Range): boolean;
}
//# sourceMappingURL=RangeUtility.d.ts.map