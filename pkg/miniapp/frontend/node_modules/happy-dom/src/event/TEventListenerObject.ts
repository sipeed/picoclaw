import Event from './Event.js';

type TEventListenerObject = {
	handleEvent(event: Event): void;
};

export default TEventListenerObject;
