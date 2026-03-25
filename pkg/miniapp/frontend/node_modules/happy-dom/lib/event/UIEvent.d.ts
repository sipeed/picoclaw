import BrowserWindow from '../window/BrowserWindow.js';
import Event from './Event.js';
import IUIEventInit from './IUIEventInit.js';
/**
 *
 */
export default class UIEvent extends Event {
    static NONE: number;
    static CAPTURING_PHASE: number;
    static AT_TARGET: number;
    static BUBBLING_PHASE: number;
    readonly detail: number;
    readonly layerX: number;
    readonly layerY: number;
    readonly pageX: number;
    readonly pageY: number;
    readonly view: BrowserWindow | null;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IUIEventInit | null);
}
//# sourceMappingURL=UIEvent.d.ts.map