import UIEvent from '../UIEvent.js';
import EventTarget from '../EventTarget.js';
import IMouseEventInit from './IMouseEventInit.js';
/**
 *
 */
export default class MouseEvent extends UIEvent {
    readonly altKey: boolean;
    readonly button: number;
    readonly buttons: number;
    readonly clientX: number;
    readonly clientY: number;
    readonly ctrlKey: boolean;
    readonly metaKey: boolean;
    readonly movementX: number;
    readonly movementY: number;
    readonly offsetX: number;
    readonly offsetY: number;
    readonly region: string;
    readonly relatedTarget: EventTarget | null;
    readonly screenX: number;
    readonly screenY: number;
    readonly shiftKey: boolean;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IMouseEventInit | null);
}
//# sourceMappingURL=MouseEvent.d.ts.map