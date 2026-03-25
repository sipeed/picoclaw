import EventTarget from '../event/EventTarget.js';
import Event from '../event/Event.js';
/**
 * Permission status.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/PermissionStatus
 */
export default class PermissionStatus extends EventTarget {
    state: 'granted' | 'denied' | 'prompt';
    onchange: ((event: Event) => void) | null;
    /**
     * Constructor.
     *
     * @param [state] State.
     */
    constructor(state?: 'granted' | 'denied' | 'prompt');
}
//# sourceMappingURL=PermissionStatus.d.ts.map