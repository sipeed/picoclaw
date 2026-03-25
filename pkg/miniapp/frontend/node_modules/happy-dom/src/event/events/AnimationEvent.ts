import Event from '../Event.js';
import IAnimationEventInit from './IAnimationEventInit.js';

/**
 *
 */
export default class AnimationEvent extends Event {
	public readonly animationName: string;
	public readonly elapsedTime: number;
	public readonly pseudoElement: string;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IAnimationEventInit | null = null) {
		super(type, eventInit);

		this.animationName = eventInit?.animationName ?? '';
		this.elapsedTime = eventInit?.elapsedTime ?? 0;
		this.pseudoElement = eventInit?.pseudoElement ?? '';
	}
}
