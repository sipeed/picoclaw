import DOMRect from '../dom/DOMRect.js';
import Node from '../nodes/node/Node.js';
/**
 * The IntersectionObserverEntry interface of the Intersection Observer API describes the intersection between the target element and its root container at a specific moment of transition.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/IntersectionObserverEntry
 */
export default class IntersectionObserverEntry {
    readonly boundingClientRect: DOMRect | null;
    readonly intersectionRatio: number;
    readonly intersectionRect: DOMRect | null;
    readonly isIntersecting: boolean;
    readonly rootBounds: DOMRect | null;
    readonly target: Node | null;
    readonly time: number;
    /**
     * Constructor.
     *
     * @param init Options to initialize the intersection observer entry.
     */
    constructor(init?: Partial<IntersectionObserverEntry>);
}
//# sourceMappingURL=IntersectionObserverEntry.d.ts.map