import DOMPointReadOnly from './DOMPointReadOnly.js';
import * as PropertySymbol from '../PropertySymbol.js';

/**
 * DOM Point.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMPoint
 */
export default class DOMPoint extends DOMPointReadOnly {
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
	 * Sets z.
	 *
	 * @param value Z.
	 */
	public set z(value: number) {
		this[PropertySymbol.z] = value;
	}

	/**
	 * Returns z.
	 *
	 * @returns Z.
	 */
	public get z(): number {
		return this[PropertySymbol.z];
	}

	/**
	 * Sets w.
	 *
	 * @param value W.
	 */
	public set w(value: number) {
		this[PropertySymbol.w] = value;
	}

	/**
	 * Returns w.
	 *
	 * @returns W.
	 */
	public get w(): number {
		return this[PropertySymbol.w];
	}
}
