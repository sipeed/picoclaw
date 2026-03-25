import IEventInit from '../IEventInit.js';

export default interface IAnimationEventInit extends IEventInit {
	animationName?: string;
	elapsedTime?: number;
	pseudoElement?: string;
}
