import ProgressEvent from '../event/events/ProgressEvent.js';
import EventTarget from '../event/EventTarget.js';
export type ProgressEventListener = (event: ProgressEvent) => void;
/**
 * References: https://xhr.spec.whatwg.org/#xmlhttprequesteventtarget.
 */
export default class XMLHttpRequestEventTarget extends EventTarget {
    onloadstart: ProgressEventListener | null;
    onprogress: ProgressEventListener | null;
    onabort: ((event: ProgressEvent) => void) | null;
    onerror: ProgressEventListener | null;
    onload: ProgressEventListener | null;
    ontimeout: ProgressEventListener | null;
    onloadend: ProgressEventListener | null;
}
//# sourceMappingURL=XMLHttpRequestEventTarget.d.ts.map