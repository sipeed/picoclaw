import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * When loading a media resource for use by an <audio> or <video> element, the TimeRanges interface is used for representing the time ranges of the media resource that have been buffered, the time ranges that have been played, and the time ranges that are seekable.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TimeRanges
 */
export default class TimeRanges {
	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 */
	constructor(illegalConstructorSymbol: symbol) {
		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}
	}

	/**
	 * Returns length.
	 */
	public get length(): number {
		return 0;
	}

	/**
	 * Returns `Symbol.toStringTag`.
	 *
	 * @returns `Symbol.toStringTag`.
	 */
	public get [Symbol.toStringTag](): string {
		return 'TimeRanges';
	}

	/**
	 * Returns `[object NodeList]`.
	 *
	 * @returns `[object NodeList]`.
	 */
	public toLocaleString(): string {
		return '[object TimeRanges]';
	}

	/**
	 * Returns `[object NodeList]`.
	 *
	 * @returns `[object NodeList]`.
	 */
	public toString(): string {
		return '[object TimeRanges]';
	}

	/**
	 * Returns start.
	 *
	 * @param _index Index.
	 * @returns Start.
	 */
	public start(_index: number): number {
		return 0;
	}

	/**
	 * Returns end.
	 *
	 * @param _index Index.
	 * @returns End.
	 */
	public end(_index: number): number {
		return 0;
	}
}
