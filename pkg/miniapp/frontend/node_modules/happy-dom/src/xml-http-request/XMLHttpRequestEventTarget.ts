import ProgressEvent from '../event/events/ProgressEvent.js';
import EventTarget from '../event/EventTarget.js';

export type ProgressEventListener = (event: ProgressEvent) => void;

/**
 * References: https://xhr.spec.whatwg.org/#xmlhttprequesteventtarget.
 */
export default class XMLHttpRequestEventTarget extends EventTarget {
	public onloadstart: ProgressEventListener | null = null;
	public onprogress: ProgressEventListener | null = null;
	public onabort: ((event: ProgressEvent) => void) | null = null;
	public onerror: ProgressEventListener | null = null;
	public onload: ProgressEventListener | null = null;
	public ontimeout: ProgressEventListener | null = null;
	public onloadend: ProgressEventListener | null = null;
}
