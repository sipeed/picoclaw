import Event from '../Event.js';
import IProgressEventInit from './IProgressEventInit.js';

/**
 *
 */
export default class ProgressEvent extends Event {
	public readonly lengthComputable: boolean;
	public readonly loaded: number;
	public readonly total: number;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IProgressEventInit | null = null) {
		super(type);

		this.lengthComputable = eventInit?.lengthComputable ?? false;
		this.loaded = eventInit?.loaded ?? 0;
		this.total = eventInit?.total ?? 0;
	}
}
