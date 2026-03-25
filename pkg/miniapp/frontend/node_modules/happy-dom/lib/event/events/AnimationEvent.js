import Event from '../Event.js';
/**
 *
 */
export default class AnimationEvent extends Event {
    animationName;
    elapsedTime;
    pseudoElement;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type, eventInit = null) {
        super(type, eventInit);
        this.animationName = eventInit?.animationName ?? '';
        this.elapsedTime = eventInit?.elapsedTime ?? 0;
        this.pseudoElement = eventInit?.pseudoElement ?? '';
    }
}
//# sourceMappingURL=AnimationEvent.js.map