import IEventInit from '../IEventInit.js';

export default interface IMediaQueryListInit extends IEventInit {
	matches?: boolean;
	media?: string;
}
