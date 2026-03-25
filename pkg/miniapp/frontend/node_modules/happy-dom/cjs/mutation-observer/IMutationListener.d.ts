import IMutationObserverInit from './IMutationObserverInit.cjs';
import MutationRecord from './MutationRecord.cjs';
export default interface IMutationListener {
    options: IMutationObserverInit;
    callback: WeakRef<(record: MutationRecord) => void>;
}
//# sourceMappingURL=IMutationListener.d.ts.map