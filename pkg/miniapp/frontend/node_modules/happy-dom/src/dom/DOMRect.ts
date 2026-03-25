import DOMRectReadOnly from './DOMRectReadOnly.js';
import * as PropertySymbol from '../PropertySymbol.js';
import IDOMRectInit from './IDOMRectInit.js';

/**
 * DOM Rect.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMRect
 */
export default class DOMRect extends DOMRectReadOnly {
	/**
	 * Sets x.
	 *
	 * @param value X.
	 */
	public set x(value: number) {
		this[PropertySymbol.x] = value;
	}

	/**
	 * Returns x.
	 *
	 * @returns X.
	 */
	public get x(): number {
		return this[PropertySymbol.x];
	}

	/**
	 * Sets y.
	 *
	 * @param value Y.
	 */
	public set y(value: number) {
		this[PropertySymbol.y] = value;
	}

	/**
	 * Returns y.
	 *
	 * @returns Y.
	 */
	public get y(): number {
		return this[PropertySymbol.y];
	}

	/**
	 * Sets width.
	 *
	 * @param value Width.
	 */
	public set width(value: number) {
		this[PropertySymbol.width] = value;
	}

	/**
	 * Returns width.
	 *
	 * @returns Width.
	 */
	public get width(): number {
		return this[PropertySymbol.width];
	}

	/**
	 * Sets height.
	 *
	 * @param value Height.
	 */
	public set height(value: number) {
		this[PropertySymbol.height] = value;
	}

	/**
	 * Returns height.
	 *
	 * @returns Height.
	 */
	public get height(): number {
		return this[PropertySymbol.height];
	}

	/**
	 * Returns a new DOMRect object.
	 *
	 * @param other
	 * @returns Cloned object.
	 */
	public static fromRect(other: IDOMRectInit): DOMRect {
		return new DOMRect(other.x, other.y, other.width, other.height);
	}
}
