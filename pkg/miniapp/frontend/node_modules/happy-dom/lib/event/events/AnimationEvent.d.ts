import Event from '../Event.js';
import IAnimationEventInit from './IAnimationEventInit.js';
/**
 *
 */
export default class AnimationEvent extends Event {
    readonly animationName: string;
    readonly elapsedTime: number;
    readonly pseudoElement: string;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IAnimationEventInit | null);
}
//# sourceMappingURL=AnimationEvent.d.ts.map