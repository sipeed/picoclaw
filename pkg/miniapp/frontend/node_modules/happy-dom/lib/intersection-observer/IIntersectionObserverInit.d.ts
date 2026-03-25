import Element from '../nodes/element/Element.js';
export default interface IIntersectionObserverInit {
    /**
     * A specific ancestor of the target element against which the intersection is to be calculated.
     */
    root?: Element;
    /**
     * A string which specifies a specific property to observe on the intersection target.
     */
    rootMargin?: string;
    /**
     * A list of thresholds, sorted in increasing numeric order, where each threshold is a ratio of intersection area to bounding box area of the target.
     */
    threshold?: number | number[];
}
//# sourceMappingURL=IIntersectionObserverInit.d.ts.map