import UIEvent from '../UIEvent.js';
import IKeyboardEventInit from './IKeyboardEventInit.js';

/**
 *
 */
export default class KeyboardEvent extends UIEvent {
	public static DOM_KEY_LOCATION_STANDARD = 0;
	public static DOM_KEY_LOCATION_LEFT = 1;
	public static DOM_KEY_LOCATION_RIGHT = 2;
	public static DOM_KEY_LOCATION_NUMPAD = 3;
	public readonly altKey: boolean;
	public readonly code: string;
	public readonly ctrlKey: boolean;
	public readonly isComposing: boolean;
	public readonly key: string;
	public readonly location: number;
	public readonly metaKey: boolean;
	public readonly repeat: boolean;
	public readonly shiftKey: boolean;

	/**
	 * @deprecated
	 */
	public readonly keyCode: number;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IKeyboardEventInit | null = null) {
		super(type, eventInit);

		this.altKey = eventInit?.altKey ?? false;
		this.code = eventInit?.code ?? '';
		this.ctrlKey = eventInit?.ctrlKey ?? false;
		this.isComposing = eventInit?.isComposing ?? false;
		this.key = eventInit?.key ?? '';
		this.location = eventInit?.location ?? 0;
		this.metaKey = eventInit?.metaKey ?? false;
		this.repeat = eventInit?.repeat ?? false;
		this.shiftKey = eventInit?.shiftKey ?? false;
		this.keyCode = eventInit?.keyCode ?? 0;
	}
}
