import DOMException from '../../exception/DOMException.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import IRequestReferrerPolicy from '../types/IRequestReferrerPolicy.js';
import IRequestRedirect from '../types/IRequestRedirect.js';
import URL from '../../url/URL.js';
import Request from '../Request.js';

const VALID_REFERRER_POLICIES = [
	'',
	'no-referrer',
	'no-referrer-when-downgrade',
	'same-origin',
	'origin',
	'strict-origin',
	'origin-when-cross-origin',
	'strict-origin-when-cross-origin',
	'unsafe-url'
];

const VALID_REDIRECTS = ['error', 'manual', 'follow'];

const SUPPORTED_SCHEMAS = ['data:', 'http:', 'https:'];

const FORBIDDEN_REQUEST_METHODS = ['TRACE', 'TRACK', 'CONNECT'];
const REQUEST_METHOD_REGEXP = /^[A-Z]+$/;

/**
 * Fetch request validation utility.
 */
export default class FetchRequestValidationUtility {
	/**
	 * Validates request method.
	 *
	 * @throws DOMException
	 * @param request Request.
	 */
	public static validateMethod(request: Request): void {
		if (!request.method || FORBIDDEN_REQUEST_METHODS.includes(request.method)) {
			throw new DOMException(
				`'${request.method || ''}' is not a valid HTTP method.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		if (!REQUEST_METHOD_REGEXP.test(request.method)) {
			throw new DOMException(
				`'${request.method}' HTTP method is unsupported.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}
	}

	/**
	 * Validates request body.
	 *
	 * @throws DOMException
	 * @param request Request.
	 */
	public static validateBody(request: Request): void {
		if (request.body && (request.method === 'GET' || request.method === 'HEAD')) {
			throw new DOMException(
				`Request with GET/HEAD method cannot have body.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}
	}

	/**
	 * Validates request URL.
	 *
	 * @throws DOMException
	 * @param url URL.
	 */
	public static validateURL(url: URL): void {
		if (url.username !== '' || url.password !== '') {
			throw new DOMException(
				`${url} is an url with embedded credentials.`,
				DOMExceptionNameEnum.notSupportedError
			);
		}
	}

	/**
	 * Validates request referrer policy.
	 *
	 * @throws DOMException
	 * @param referrerPolicy Referrer policy.
	 */
	public static validateReferrerPolicy(referrerPolicy: IRequestReferrerPolicy): void {
		if (!VALID_REFERRER_POLICIES.includes(referrerPolicy)) {
			throw new DOMException(
				`Invalid referrer policy "${referrerPolicy}".`,
				DOMExceptionNameEnum.syntaxError
			);
		}
	}

	/**
	 * Validates request redirect.
	 *
	 * @throws DOMException
	 * @param redirect Redirect.
	 */
	public static validateRedirect(redirect: IRequestRedirect): void {
		if (!VALID_REDIRECTS.includes(redirect)) {
			throw new DOMException(`Invalid redirect "${redirect}".`, DOMExceptionNameEnum.syntaxError);
		}
	}

	/**
	 * Validates request redirect.
	 *
	 * @throws DOMException
	 * @param request
	 * @param redirect Redirect.
	 */
	public static validateSchema(request: Request): void {
		if (!SUPPORTED_SCHEMAS.includes(request[PropertySymbol.url].protocol)) {
			throw new DOMException(
				`Failed to fetch from "${request.url}": URL scheme "${request[
					PropertySymbol.url
				].protocol.replace(/:$/, '')}" is not supported.`,
				DOMExceptionNameEnum.notSupportedError
			);
		}
	}
}
