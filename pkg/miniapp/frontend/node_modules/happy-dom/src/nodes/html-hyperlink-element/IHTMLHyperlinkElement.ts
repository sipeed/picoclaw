/**
 * HTMLHyperlinkElementUtils.
 *
 * Reference:
 * https://html.spec.whatwg.org/multipage/links.html#htmlhyperlinkelementutils.
 */
export default interface IHTMLHyperlinkElement {
	readonly origin: string;
	href: string;
	protocol: string;
	username: string;
	password: string;
	host: string;
	hostname: string;
	port: string;
	pathname: string;
	search: string;
	hash: string;
}
