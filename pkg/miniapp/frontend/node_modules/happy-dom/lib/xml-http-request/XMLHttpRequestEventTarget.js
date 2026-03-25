import EventTarget from '../event/EventTarget.js';
/**
 * References: https://xhr.spec.whatwg.org/#xmlhttprequesteventtarget.
 */
export default class XMLHttpRequestEventTarget extends EventTarget {
    onloadstart = null;
    onprogress = null;
    onabort = null;
    onerror = null;
    onload = null;
    ontimeout = null;
    onloadend = null;
}
//# sourceMappingURL=XMLHttpRequestEventTarget.js.map