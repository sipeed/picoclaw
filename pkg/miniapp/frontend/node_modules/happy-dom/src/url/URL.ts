import { URL as NodeJSURL } from 'url';
import * as PropertySymbol from '../PropertySymbol.js';
import { Blob as NodeJSBlob } from 'buffer';
import Blob from '../file/Blob.js';

/**
 * URL.
 */
export default class URL extends NodeJSURL {
	/**
	 * Creates a string containing a URL representing the object given in the parameter.
	 *
	 * @param object Object.
	 * @returns URL.
	 */
	public static override createObjectURL(object: NodeJSBlob | Blob): string {
		if (object instanceof Blob) {
			const blob = new NodeJSBlob([object[PropertySymbol.buffer]], { type: object.type });
			return super.createObjectURL(blob);
		}
		return super.createObjectURL(object);
	}
}
