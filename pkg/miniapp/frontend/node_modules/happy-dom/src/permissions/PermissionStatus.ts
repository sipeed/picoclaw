import EventTarget from '../event/EventTarget.js';
import Event from '../event/Event.js';

/**
 * Permission status.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/PermissionStatus
 */
export default class PermissionStatus extends EventTarget {
	public state: 'granted' | 'denied' | 'prompt';
	public onchange: ((event: Event) => void) | null = null;

	/**
	 * Constructor.
	 *
	 * @param [state] State.
	 */
	constructor(state: 'granted' | 'denied' | 'prompt' = 'granted') {
		super();
		this.state = state;
	}
}
