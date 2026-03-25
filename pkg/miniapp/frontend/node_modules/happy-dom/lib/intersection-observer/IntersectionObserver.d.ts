import IntersectionObserverEntry from './IntersectionObserverEntry.js';
import IIntersectionObserverInit from './IIntersectionObserverInit.js';
import Element from '../nodes/element/Element.js';
/**
 * The IntersectionObserver interface of the Intersection Observer API provides a way to asynchronously observe changes in the intersection of a target element with an ancestor element or with a top-level document's viewport.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/IntersectionObserver
 */
export default class IntersectionObserver {
    #private;
    /**
     * Constructor.
     *
     * @param callback Callback.
     * @param options Options.
     */
    constructor(callback: (entries: IntersectionObserverEntry[], observer: IntersectionObserver) => void, options?: IIntersectionObserverInit);
    /**
     * Starts observing.
     *
     * @param _target Target.
     */
    observe(_target: Element): void;
    /**
     * Disconnects.
     */
    disconnect(): void;
    /**
     * Unobserves an element.
     *
     * @param _target Target.
     */
    unobserve(_target: Element): void;
    /**
     * Returns an array of IntersectionObserverEntry objects for all observed targets.
     *
     * @returns Records.
     */
    takeRecords(): IntersectionObserverEntry[];
}
//# sourceMappingURL=IntersectionObserver.d.ts.map