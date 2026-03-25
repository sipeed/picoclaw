import IEventInit from '../IEventInit.js';

export default interface ICustomEventInit extends IEventInit {
	detail?: object | null;
}
