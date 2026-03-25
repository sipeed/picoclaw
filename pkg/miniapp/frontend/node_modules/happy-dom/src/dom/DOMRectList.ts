import DOMRect from './DOMRect.js';
import * as PropertySymbol from '../PropertySymbol.js';

/**
 * DOM Rect List.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMRectList
 */
export default class DOMRectList extends Array<DOMRect> {
	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 */
	constructor(illegalConstructorSymbol: symbol) {
		super();
		// "illegalConstructorSymbol" can be "1" when calling the "splice()" method
		if (
			<number>(<unknown>illegalConstructorSymbol) !== 1 &&
			illegalConstructorSymbol !== PropertySymbol.illegalConstructor
		) {
			throw new TypeError('Illegal constructor');
		}
	}

	/**
	 * Returns item by index.
	 *
	 * @param index Index.
	 */
	public item(index: number): DOMRect {
		return this[index] ?? null;
	}
}
