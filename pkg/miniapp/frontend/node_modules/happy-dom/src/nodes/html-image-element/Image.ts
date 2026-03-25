import HTMLImageElement from './HTMLImageElement.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import NamespaceURI from '../../config/NamespaceURI.js';

/**
 * Image as constructor.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/HTMLImageElement/Image.
 */
export default class Image extends HTMLImageElement {
	public [PropertySymbol.tagName] = 'IMG';
	public [PropertySymbol.localName] = 'img';
	public [PropertySymbol.namespaceURI] = NamespaceURI.html;

	/**
	 * Constructor.
	 *
	 * @param [width] Width.
	 * @param [height] Height.
	 */
	constructor(width: number = null, height: number = null) {
		super();

		if (width !== null) {
			this.width = width;
		}

		if (height !== null) {
			this.height = height;
		}
	}
}
