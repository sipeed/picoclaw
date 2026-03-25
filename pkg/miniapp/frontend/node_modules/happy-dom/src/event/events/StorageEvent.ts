import Event from '../Event.js';
import IStorageEventInit from './IStorageEventInit.js';
import Storage from '../../storage/Storage.js';

/**
 *
 */
export default class StorageEvent extends Event {
	public readonly key: string | null;
	public readonly oldValue: string | null;
	public readonly newValue: string | null;
	public readonly url: string;
	public readonly storageArea: Storage | null;

	/**
	 * Constructor.
	 *
	 * @param type Event type.
	 * @param [eventInit] Event init.
	 */
	constructor(type: string, eventInit: IStorageEventInit | null = null) {
		super(type);

		this.key = eventInit?.key ?? null;
		this.oldValue = eventInit?.oldValue ?? null;
		this.newValue = eventInit?.newValue ?? null;
		this.url = eventInit?.url ?? '';
		this.storageArea = eventInit?.storageArea ?? null;
	}
}
