import IEventInit from '../IEventInit.js';
import Storage from '../../storage/Storage.js';
export default interface IStorageEventInit extends IEventInit {
    key?: string;
    oldValue?: string;
    newValue?: string;
    url?: string;
    storageArea?: Storage;
}
//# sourceMappingURL=IStorageEventInit.d.ts.map