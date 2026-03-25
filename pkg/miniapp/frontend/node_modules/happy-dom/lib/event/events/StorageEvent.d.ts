import Event from '../Event.js';
import IStorageEventInit from './IStorageEventInit.js';
import Storage from '../../storage/Storage.js';
/**
 *
 */
export default class StorageEvent extends Event {
    readonly key: string | null;
    readonly oldValue: string | null;
    readonly newValue: string | null;
    readonly url: string;
    readonly storageArea: Storage | null;
    /**
     * Constructor.
     *
     * @param type Event type.
     * @param [eventInit] Event init.
     */
    constructor(type: string, eventInit?: IStorageEventInit | null);
}
//# sourceMappingURL=StorageEvent.d.ts.map