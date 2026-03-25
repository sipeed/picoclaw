/**
 * The IntersectionObserverEntry interface of the Intersection Observer API describes the intersection between the target element and its root container at a specific moment of transition.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/IntersectionObserverEntry
 */
export default class IntersectionObserverEntry {
    boundingClientRect = null;
    intersectionRatio = 0;
    intersectionRect = null;
    isIntersecting = false;
    rootBounds = null;
    target = null;
    time = 0;
    /**
     * Constructor.
     *
     * @param init Options to initialize the intersection observer entry.
     */
    constructor(init) {
        Object.assign(this, init);
    }
}
//# sourceMappingURL=IntersectionObserverEntry.js.map