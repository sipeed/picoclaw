import DOMExceptionNameEnum from './DOMExceptionNameEnum.js';

/**
 * DOM Exception.
 *
 * Reference:
 * https://developer.mozilla.org/en-US/docs/Web/API/DOMException/DOMException.
 */
export default class DOMException extends Error {
	/**
	 * Constructor.
	 *
	 * @param message Message.
	 * @param name Name.
	 */
	constructor(message: string, name: string = null) {
		super(message);

		this.name = name || DOMExceptionNameEnum.domException;
	}
}
