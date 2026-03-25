import NamespaceURI from '../../config/NamespaceURI.js';
import HTMLAudioElement from './HTMLAudioElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
/**
 * Image as constructor.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLAudioElement/Audio.
 */
export default class Audio extends HTMLAudioElement {
    [PropertySymbol.tagName] = 'AUDIO';
    [PropertySymbol.localName] = 'audio';
    [PropertySymbol.namespaceURI] = NamespaceURI.html;
    /**
     * Constructor.
     *
     * @param [url] source URL.
     */
    constructor(url = null) {
        super();
        if (url !== null) {
            this.src = url;
        }
    }
}
//# sourceMappingURL=Audio.js.map