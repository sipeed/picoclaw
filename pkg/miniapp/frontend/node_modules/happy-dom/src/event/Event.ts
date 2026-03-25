import IEventInit from './IEventInit.js';
import * as PropertySymbol from '../PropertySymbol.js';
import Node from '../nodes/node/Node.js';
import BrowserWindow from '../window/BrowserWindow.js';
import ShadowRoot from '../nodes/shadow-root/ShadowRoot.js';
import EventTarget from './EventTarget.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import EventPhaseEnum from './EventPhaseEnum.js';
import Document from '../nodes/document/Document.js';

/**
 * Event.
 */
export default class Event {
	public NONE = EventPhaseEnum.none;
	public CAPTURING_PHASE = EventPhaseEnum.capturing;
	public AT_TARGET = EventPhaseEnum.atTarget;
	public BUBBLING_PHASE = EventPhaseEnum.bubbling;

	public [PropertySymbol.composed] = false;
	public [PropertySymbol.bubbles] = false;
	public [PropertySymbol.cancelable] = false;
	public [PropertySymbol.defaultPrevented] = false;
	public [PropertySymbol.eventPhase] = EventPhaseEnum.none;
	public [PropertySymbol.timeStamp] = performance.now();
	public [PropertySymbol.type]: string;

	public [PropertySymbol.dispatching] = false;
	public [PropertySymbol.immediatePropagationStopped] = false;
	public [PropertySymbol.propagationStopped] = false;
	public [PropertySymbol.target]: EventTarget = null;
	public [PropertySymbol.currentTarget]: EventTarget = null;
	public [PropertySymbol.isInPassiveEventListener] = false;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IEventInit | null = null) {
		this[PropertySymbol.type] = type;
		this[PropertySymbol.bubbles] = eventInit?.bubbles ?? false;
		this[PropertySymbol.cancelable] = eventInit?.cancelable ?? false;
		this[PropertySymbol.composed] = eventInit?.composed ?? false;
	}

	/**
	 * Returns composed.
	 *
	 * @returns Composed.
	 */
	public get composed(): boolean {
		return this[PropertySymbol.composed];
	}

	/**
	 * Returns bubbles.
	 *
	 * @returns Bubbles.
	 */
	public get bubbles(): boolean {
		return this[PropertySymbol.bubbles];
	}

	/**
	 * Returns cancelable.
	 *
	 * @returns Cancelable.
	 */
	public get cancelable(): boolean {
		return this[PropertySymbol.cancelable];
	}

	/**
	 * Returns defaultPrevented.
	 *
	 * @returns Default prevented.
	 */
	public get defaultPrevented(): boolean {
		return this[PropertySymbol.defaultPrevented];
	}

	/**
	 * Returns eventPhase.
	 *
	 * @returns Event phase.
	 */
	public get eventPhase(): EventPhaseEnum {
		return this[PropertySymbol.eventPhase];
	}

	/**
	 * Returns timeStamp.
	 *
	 * @returns Time stamp.
	 */
	public get timeStamp(): number {
		return this[PropertySymbol.timeStamp];
	}

	/**
	 * Returns type.
	 *
	 * @returns Type.
	 */
	public get type(): string {
		return this[PropertySymbol.type];
	}

	/**
	 * Returns target.
	 *
	 * @returns Target.
	 */
	public get target(): EventTarget {
		return this[PropertySymbol.target];
	}

	/**
	 * Returns target.
	 *
	 * @returns Target.
	 */
	public get currentTarget(): EventTarget {
		return this[PropertySymbol.currentTarget];
	}

	/**
	 * Returns "true" if propagation has been stopped.
	 *
	 * @returns "true" if propagation has been stopped.
	 */
	public get cancelBubble(): boolean {
		return this[PropertySymbol.propagationStopped];
	}

	/**
	 * Returns composed path.
	 *
	 * @returns Composed path.
	 */
	public composedPath(): EventTarget[] {
		if (!this[PropertySymbol.target]) {
			return [];
		}

		const composedPath = [];
		let eventTarget: Node | ShadowRoot | BrowserWindow = <Node | ShadowRoot>(
			(<unknown>this[PropertySymbol.target])
		);

		while (eventTarget) {
			composedPath.push(eventTarget);

			if ((<Node>(<unknown>eventTarget)).parentNode) {
				eventTarget = (<Node>(<unknown>eventTarget)).parentNode;
			} else if (
				this[PropertySymbol.composed] &&
				(<Node>eventTarget)[PropertySymbol.nodeType] === NodeTypeEnum.documentFragmentNode &&
				(<ShadowRoot>eventTarget).host
			) {
				eventTarget = (<ShadowRoot>eventTarget).host;
			} else if (
				(<Node>eventTarget)[PropertySymbol.nodeType] === NodeTypeEnum.documentNode &&
				// The "load" event is a special case. It should not bubble up to the window.
				this[PropertySymbol.type] !== 'load'
			) {
				eventTarget = (<Document>(<unknown>eventTarget))[PropertySymbol.window];
			} else {
				break;
			}
		}

		return composedPath;
	}

	/**
	 * Init event.
	 *
	 * @deprecated
	 * @param type Type.
	 * @param [bubbles=false] "true" if it bubbles.
	 * @param [cancelable=false] "true" if it cancelable.
	 */
	public initEvent(type: string, bubbles = false, cancelable = false): void {
		this[PropertySymbol.type] = type;
		this[PropertySymbol.bubbles] = bubbles;
		this[PropertySymbol.cancelable] = cancelable;
	}

	/**
	 * Prevents default.
	 */
	public preventDefault(): void {
		if (!this[PropertySymbol.isInPassiveEventListener] && this.cancelable) {
			this[PropertySymbol.defaultPrevented] = true;
		}
	}

	/**
	 * Stops immediate propagation.
	 */
	public stopImmediatePropagation(): void {
		this[PropertySymbol.immediatePropagationStopped] = true;
	}

	/**
	 * Stops propagation.
	 */
	public stopPropagation(): void {
		this[PropertySymbol.propagationStopped] = true;
	}
}
