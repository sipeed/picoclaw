import Event from '../Event.js';
import ICustomEventInit from './ICustomEventInit.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 *
 */
export default class CustomEvent extends Event {
	// eslint-disable-next-line @typescript-eslint/no-explicit-any
	public [PropertySymbol.detail]: any;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: ICustomEventInit | null = null) {
		super(type, eventInit);

		this[PropertySymbol.detail] = eventInit?.detail ?? null;
	}

	/**
	 * Returns detail.
	 *
	 * @returns Detail.
	 */
	public get detail(): any {
		return this[PropertySymbol.detail];
	}

	/**
	 * Init event.
	 *
	 * @deprecated
	 * @param type Type.
	 * @param [bubbles=false] "true" if it bubbles.
	 * @param [cancelable=false] "true" if it cancelable.
	 * @param [detail=null] Custom event detail.
	 */
	public initCustomEvent(
		type: string,
		bubbles = false,
		cancelable = false,
		detail: object = null
	): void {
		this[PropertySymbol.type] = type;
		this[PropertySymbol.bubbles] = bubbles;
		this[PropertySymbol.cancelable] = cancelable;
		this[PropertySymbol.detail] = detail;
	}
}
