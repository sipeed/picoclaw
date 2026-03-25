import IMutationObserverInit from './IMutationObserverInit.js';
import MutationObserver from './MutationObserver.js';
import MutationRecord from './MutationRecord.js';
import Node from '../nodes/node/Node.js';
import BrowserWindow from '../window/BrowserWindow.js';
import IMutationListener from './IMutationListener.js';
/**
 * Mutation Observer Listener.
 */
export default class MutationObserverListener {
    #private;
    readonly target: Node;
    options: IMutationObserverInit;
    mutationListener: IMutationListener;
    /**
     * Constructor.
     *
     * @param init Options.
     * @param init.window Window.
     * @param init.options Options.
     * @param init.target Target.
     * @param init.observer Observer.
     * @param init.callback Callback.
     */
    constructor(init: {
        window: BrowserWindow;
        options: IMutationObserverInit;
        target: Node;
        observer: MutationObserver;
        callback: (record: MutationRecord[], observer: MutationObserver) => void;
    });
    /**
     * Reports mutations.
     *
     * @param record Record.
     */
    report(record: MutationRecord): void;
    /**
     * Destroys the listener.
     */
    takeRecords(): MutationRecord[];
    /**
     * Destroys the listener.
     */
    destroy(): void;
}
//# sourceMappingURL=MutationObserverListener.d.ts.map