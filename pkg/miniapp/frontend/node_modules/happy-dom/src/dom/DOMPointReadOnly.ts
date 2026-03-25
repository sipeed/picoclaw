import * as PropertySymbol from '../PropertySymbol.js';
import IDOMPointInit from './IDOMPointInit.js';

/**
 * DOM Point Readonly.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/DOMPointReadOnly
 */
export default class DOMPointReadOnly implements IDOMPointInit {
	protected [PropertySymbol.x]: number = 0;
	protected [PropertySymbol.y]: number = 0;
	protected [PropertySymbol.z]: number = 0;
	protected [PropertySymbol.w]: number = 1;

	/**
	 * Constructor.
	 *
	 * @param [x] X position.
	 * @param [y] Y position.
	 * @param [z] Width.
	 * @param [w] Height.
	 */
	constructor(x?: number | null, y?: number | null, z?: number | null, w?: number | null) {
		this[PropertySymbol.x] = x !== undefined && x !== null ? Number(x) : 0;
		this[PropertySymbol.y] = y !== undefined && y !== null ? Number(y) : 0;
		this[PropertySymbol.z] = z !== undefined && z !== null ? Number(z) : 0;
		this[PropertySymbol.w] = w !== undefined && w !== null ? Number(w) : 1;
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
	 * Returns y.
	 *
	 * @returns Y.
	 */
	public get y(): number {
		return this[PropertySymbol.y];
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
	 * Returns w.
	 *
	 * @returns W.
	 */
	public get w(): number {
		return this[PropertySymbol.w];
	}

	/**
	 * Returns the JSON representation of the object.
	 *
	 * @returns JSON representation.
	 */
	public toJSON(): object {
		return {
			x: this.x,
			y: this.y,
			z: this.z,
			w: this.w
		};
	}

	/**
	 * Returns a new DOMPointReadOnly object.
	 *
	 * @param [otherPoint] Other point.
	 * @returns Cloned object.
	 */
	public static fromPoint(otherPoint?: IDOMPointInit): DOMPointReadOnly {
		if (!otherPoint) {
			return new this();
		}
		return new this(
			otherPoint.x ?? null,
			otherPoint.y ?? null,
			otherPoint.z ?? null,
			otherPoint.w ?? null
		);
	}
}
