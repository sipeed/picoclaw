import Event from '../../event/Event.js';
import HTMLElement from '../html-element/HTMLElement.js';
/**
 * HTMLBodyElement
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/HTMLBodyElement
 */
export default class HTMLBodyElement extends HTMLElement {
	// Events
	public onafterprint: ((event: Event) => void) | null = null;
	public onbeforeprint: ((event: Event) => void) | null = null;
	public onbeforeunload: ((event: Event) => void) | null = null;
	public ongamepadconnected: ((event: Event) => void) | null = null;
	public ongamepaddisconnected: ((event: Event) => void) | null = null;
	public onhashchange: ((event: Event) => void) | null = null;
	public onlanguagechange: ((event: Event) => void) | null = null;
	public onmessage: ((event: Event) => void) | null = null;
	public onmessageerror: ((event: Event) => void) | null = null;
	public onoffline: ((event: Event) => void) | null = null;
	public ononline: ((event: Event) => void) | null = null;
	public onpagehide: ((event: Event) => void) | null = null;
	public onpageshow: ((event: Event) => void) | null = null;
	public onpopstate: ((event: Event) => void) | null = null;
	public onrejectionhandled: ((event: Event) => void) | null = null;
	public onstorage: ((event: Event) => void) | null = null;
	public onunhandledrejection: ((event: Event) => void) | null = null;
	public onunload: ((event: Event) => void) | null = null;
}
