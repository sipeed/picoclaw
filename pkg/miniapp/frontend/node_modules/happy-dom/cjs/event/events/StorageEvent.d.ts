import Event from '../Event.cjs';
import IStorageEventInit from './IStorageEventInit.cjs';
import Storage from '../../storage/Storage.cjs';
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