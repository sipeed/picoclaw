import DataTransfer from '../DataTransfer.js';
import UIEvent from '../UIEvent.js';
import IInputEventInit from './IInputEventInit.js';

/**
 *
 */
export default class InputEvent extends UIEvent {
	public readonly data: string;
	public readonly dataTransfer: DataTransfer;
	public readonly inputType: string;
	public readonly isComposing: boolean;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IInputEventInit | null = null) {
		super(type, eventInit);

		this.data = eventInit?.data ?? '';
		this.dataTransfer = eventInit?.dataTransfer ?? null;
		this.inputType = eventInit?.inputType ?? '';
		this.isComposing = eventInit?.isComposing ?? false;
	}
}
