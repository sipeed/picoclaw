import IntersectionObserverEntry from './IntersectionObserverEntry.js';
import IIntersectionObserverInit from './IIntersectionObserverInit.js';
import Element from '../nodes/element/Element.js';

/**
 * The IntersectionObserver interface of the Intersection Observer API provides a way to asynchronously observe changes in the intersection of a target element with an ancestor element or with a top-level document's viewport.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/IntersectionObserver
 */
export default class IntersectionObserver {
	// @ts-ignore
	#callback: (entries: IntersectionObserverEntry[], observer: IntersectionObserver) => void;
	// @ts-ignore
	#options: IIntersectionObserverInit;

	/**
	 * Constructor.
	 *
	 * @param callback Callback.
	 * @param options Options.
	 */
	constructor(
		callback: (entries: IntersectionObserverEntry[], observer: IntersectionObserver) => void,
		options?: IIntersectionObserverInit
	) {
		this.#callback = callback;
		this.#options = options || {};
	}

	/**
	 * Starts observing.
	 *
	 * @param _target Target.
	 */
	public observe(_target: Element): void {
		// TODO: Implement
	}

	/**
	 * Disconnects.
	 */
	public disconnect(): void {
		// TODO: Implement
	}

	/**
	 * Unobserves an element.
	 *
	 * @param _target Target.
	 */
	public unobserve(_target: Element): void {
		// TODO: Implement
	}

	/**
	 * Returns an array of IntersectionObserverEntry objects for all observed targets.
	 *
	 * @returns Records.
	 */
	public takeRecords(): IntersectionObserverEntry[] {
		// TODO: Implement
		return [];
	}
}
