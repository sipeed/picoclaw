import IEventInit from '../IEventInit.cjs';
import Storage from '../../storage/Storage.cjs';
export default interface IStorageEventInit extends IEventInit {
    key?: string;
    oldValue?: string;
    newValue?: string;
    url?: string;
    storageArea?: Storage;
}
//# sourceMappingURL=IStorageEventInit.d.ts.map