"use strict";
Object.defineProperty(exports, "__esModule", { value: true });
/**
 * The IntersectionObserver interface of the Intersection Observer API provides a way to asynchronously observe changes in the intersection of a target element with an ancestor element or with a top-level document's viewport.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/IntersectionObserver
 */
class IntersectionObserver {
    // @ts-ignore
    #callback;
    // @ts-ignore
    #options;
    /**
     * Constructor.
     *
     * @param callback Callback.
     * @param options Options.
     */
    constructor(callback, options) {
        this.#callback = callback;
        this.#options = options || {};
    }
    /**
     * Starts observing.
     *
     * @param _target Target.
     */
    observe(_target) {
        // TODO: Implement
    }
    /**
     * Disconnects.
     */
    disconnect() {
        // TODO: Implement
    }
    /**
     * Unobserves an element.
     *
     * @param _target Target.
     */
    unobserve(_target) {
        // TODO: Implement
    }
    /**
     * Returns an array of IntersectionObserverEntry objects for all observed targets.
     *
     * @returns Records.
     */
    takeRecords() {
        // TODO: Implement
        return [];
    }
}
exports.default = IntersectionObserver;
//# sourceMappingURL=IntersectionObserver.cjs.map