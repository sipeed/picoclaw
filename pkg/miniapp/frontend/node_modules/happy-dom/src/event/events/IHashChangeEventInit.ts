import IEventInit from '../IEventInit.js';

export default interface IHashChangeEventInit extends IEventInit {
	newURL?: string;
	oldURL?: string;
}
