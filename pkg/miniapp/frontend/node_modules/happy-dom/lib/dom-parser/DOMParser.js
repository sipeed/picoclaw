import * as PropertySymbol from '../PropertySymbol.js';
import XMLParser from '../xml-parser/XMLParser.js';
import HTMLParser from '../html-parser/HTMLParser.js';
/**
 * DOM parser.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DOMParser.
 */
export default class DOMParser {
    /**
     * Parses HTML and returns a root element.
     *
     * @param string HTML data.
     * @param mimeType Mime type.
     * @returns Root element.
     */
    parseFromString(string, mimeType) {
        if (!mimeType) {
            throw new this[PropertySymbol.window].DOMException('Second parameter "mimeType" is mandatory.');
        }
        const window = this[PropertySymbol.window];
        switch (mimeType) {
            case 'text/html':
                const newDocument = new window.HTMLDocument();
                newDocument[PropertySymbol.defaultView] = window;
                return new HTMLParser(this[PropertySymbol.window]).parse(string, newDocument);
            case 'image/svg+xml':
            case 'text/xml':
            case 'application/xml':
            case 'application/xhtml+xml':
                return new XMLParser(this[PropertySymbol.window]).parse(string);
            default:
                throw new window.DOMException(`Unknown mime type "${mimeType}".`);
        }
    }
}
//# sourceMappingURL=DOMParser.js.map