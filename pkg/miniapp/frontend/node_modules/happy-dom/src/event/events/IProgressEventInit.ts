import IEventInit from '../IEventInit.js';

export default interface IProgressEventInit extends IEventInit {
	lengthComputable?: boolean;
	loaded?: number;
	total?: number;
}
