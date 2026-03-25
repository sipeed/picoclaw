import DOMRect from '../dom/DOMRect.js';
import Node from '../nodes/node/Node.js';

/**
 * The IntersectionObserverEntry interface of the Intersection Observer API describes the intersection between the target element and its root container at a specific moment of transition.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/IntersectionObserverEntry
 */
export default class IntersectionObserverEntry {
	public readonly boundingClientRect: DOMRect | null = null;
	public readonly intersectionRatio: number = 0;
	public readonly intersectionRect: DOMRect | null = null;
	public readonly isIntersecting: boolean = false;
	public readonly rootBounds: DOMRect | null = null;
	public readonly target: Node | null = null;
	public readonly time: number = 0;

	/**
	 * Constructor.
	 *
	 * @param init Options to initialize the intersection observer entry.
	 */
	constructor(init?: Partial<IntersectionObserverEntry>) {
		Object.assign(this, init);
	}
}
