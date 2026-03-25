import Event from '../Event.cjs';
import IAnimationEventInit from './IAnimationEventInit.cjs';
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