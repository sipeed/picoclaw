import Document from '../document/Document.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * Document.
 */
export default class XMLDocument extends Document {
	// Internal properties
	public [PropertySymbol.contentType]: string = 'application/xml';
}
