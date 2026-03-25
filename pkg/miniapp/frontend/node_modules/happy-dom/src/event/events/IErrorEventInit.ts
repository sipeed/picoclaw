import IEventInit from '../IEventInit.js';
export default interface IErrorEventInit extends IEventInit {
	message?: string;
	filename?: string;
	lineno?: number;
	colno?: number;
	error?: Error;
}
