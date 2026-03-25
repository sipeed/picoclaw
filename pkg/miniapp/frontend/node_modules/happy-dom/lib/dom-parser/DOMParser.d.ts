import Document from '../nodes/document/Document.js';
import * as PropertySymbol from '../PropertySymbol.js';
import BrowserWindow from '../window/BrowserWindow.js';
/**
 * DOM parser.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DOMParser.
 */
export default class DOMParser {
    protected [PropertySymbol.window]: BrowserWindow;
    /**
     * Parses HTML and returns a root element.
     *
     * @param string HTML data.
     * @param mimeType Mime type.
     * @returns Root element.
     */
    parseFromString(string: string, mimeType: string): Document;
}
//# sourceMappingURL=DOMParser.d.ts.map