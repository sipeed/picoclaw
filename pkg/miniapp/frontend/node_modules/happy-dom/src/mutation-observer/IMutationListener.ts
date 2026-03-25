import IMutationObserverInit from './IMutationObserverInit.js';
import MutationRecord from './MutationRecord.js';

export default interface IMutationListener {
	options: IMutationObserverInit;
	callback: WeakRef<(record: MutationRecord) => void>;
}
