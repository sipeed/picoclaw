import UIEvent from '../UIEvent.js';
import IErrorEventInit from './IErrorEventInit.js';

/**
 *
 */
export default class ErrorEvent extends UIEvent {
	public readonly message: string;
	public readonly filename: string;
	public readonly lineno: number;
	public readonly colno: number;
	public readonly error: Error | null;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IErrorEventInit | null = null) {
		super(type, eventInit);

		this.message = eventInit?.message ?? '';
		this.filename = eventInit?.filename ?? '';
		this.lineno = eventInit?.lineno ?? 0;
		this.colno = eventInit?.colno ?? 0;
		this.error = eventInit?.error ?? null;
	}
}
