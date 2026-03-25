import IRequestInit from './types/IRequestInit.js';
import * as PropertySymbol from '../PropertySymbol.js';
import IRequestInfo from './types/IRequestInfo.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import URL from '../url/URL.js';
import FS from 'fs';
import Path from 'path';
import Request from './Request.js';
import IBrowserFrame from '../browser/types/IBrowserFrame.js';
import BrowserWindow from '../window/BrowserWindow.js';
import ChildProcess from 'child_process';
import ISyncResponse from './types/ISyncResponse.js';
import Headers from './Headers.js';
import CachedResponseStateEnum from './cache/response/CachedResponseStateEnum.js';
import FetchRequestReferrerUtility from './utilities/FetchRequestReferrerUtility.js';
import FetchRequestValidationUtility from './utilities/FetchRequestValidationUtility.js';
import DataURIParser from './data-uri/DataURIParser.js';
import SyncFetchScriptBuilder from './utilities/SyncFetchScriptBuilder.js';
import FetchRequestHeaderUtility from './utilities/FetchRequestHeaderUtility.js';
import FetchResponseHeaderUtility from './utilities/FetchResponseHeaderUtility.js';
import Zlib from 'zlib';
import FetchResponseRedirectUtility from './utilities/FetchResponseRedirectUtility.js';
import FetchCORSUtility from './utilities/FetchCORSUtility.js';
import Fetch from './Fetch.js';
import IFetchInterceptor from './types/IFetchInterceptor.js';
import VirtualServerUtility from './utilities/VirtualServerUtility.js';

interface ISyncHTTPResponse {
	error: string;
	incomingMessage: {
		statusCode: number;
		statusMessage: string;
		rawHeaders: string[];
		data: string;
	};
}

/**
 * Handles synchrounous fetch requests.
 */
export default class SyncFetch {
	private request: Request;
	private redirectCount = 0;
	private disableCache: boolean;
	private disableSameOriginPolicy: boolean;
	private interceptor?: IFetchInterceptor;
	#browserFrame: IBrowserFrame;
	#window: BrowserWindow;
	#unfilteredHeaders: Headers | null = null;

	/**
	 * Constructor.
	 *
	 * @param options Options.
	 * @param options.browserFrame Browser frame.
	 * @param options.window Window.
	 * @param options.url URL.
	 * @param [options.init] Init.
	 * @param [options.redirectCount] Redirect count.
	 * @param [options.contentType] Content Type.
	 * @param [options.disableCache] Disables the use of cached responses. It will still store the response in the cache.
	 * @param [options.disableSameOriginPolicy] Disables the Same-Origin policy.
	 * @param [options.unfilteredHeaders] Unfiltered headers - necessary for preflight requests.
	 */
	constructor(options: {
		browserFrame: IBrowserFrame;
		window: BrowserWindow;
		url: IRequestInfo;
		init?: IRequestInit;
		redirectCount?: number;
		contentType?: string;
		disableCache?: boolean;
		disableSameOriginPolicy?: boolean;
		unfilteredHeaders?: Headers;
	}) {
		this.#browserFrame = options.browserFrame;
		this.#window = options.window;
		this.#unfilteredHeaders = options.unfilteredHeaders ?? null;
		this.request =
			typeof options.url === 'string' || options.url instanceof URL
				? new options.window.Request(options.url, options.init)
				: <Request>options.url;
		if (options.contentType) {
			(<string>this.request[PropertySymbol.contentType]) = options.contentType;
		}
		this.redirectCount = options.redirectCount ?? 0;
		this.disableCache = options.disableCache ?? false;
		this.disableSameOriginPolicy =
			options.disableSameOriginPolicy ??
			this.#browserFrame.page.context.browser.settings.fetch.disableSameOriginPolicy ??
			false;
		this.interceptor = this.#browserFrame.page.context.browser.settings.fetch.interceptor;
	}

	/**
	 * Sends request.
	 *
	 * @returns Response.
	 */
	public send(): ISyncResponse {
		FetchRequestReferrerUtility.prepareRequest(new URL(this.#window.location.href), this.request);

		const beforeRequestResponse = this.interceptor?.beforeSyncRequest
			? this.interceptor.beforeSyncRequest({
					request: this.request,
					window: this.#window
				})
			: undefined;
		if (typeof beforeRequestResponse === 'object') {
			return beforeRequestResponse;
		}

		FetchRequestValidationUtility.validateSchema(this.request);

		const virtualServerResponse = this.getVirtualServerResponse();

		if (virtualServerResponse) {
			return virtualServerResponse;
		}

		if (this.request.signal.aborted) {
			throw new this.#window.DOMException(
				'The operation was aborted.',
				DOMExceptionNameEnum.abortError
			);
		}

		if (this.request[PropertySymbol.url].protocol === 'data:') {
			const result = DataURIParser.parse(this.request.url);
			const response = {
				status: 200,
				statusText: 'OK',
				ok: true,
				url: this.request.url,
				redirected: false,
				headers: new Headers({ 'Content-Type': result.type }),
				body: result.buffer
			};
			const interceptedResponse = this.interceptor?.afterSyncResponse
				? this.interceptor.afterSyncResponse({
						window: this.#window,
						response,
						request: this.request
					})
				: undefined;
			return typeof interceptedResponse === 'object' ? interceptedResponse : response;
		}

		// Security check for "https" to "http" requests.
		if (
			this.request[PropertySymbol.url].protocol === 'http:' &&
			this.#window.location.protocol === 'https:'
		) {
			throw new this.#window.DOMException(
				`Mixed Content: The page at '${
					this.#window.location.href
				}' was loaded over HTTPS, but requested an insecure XMLHttpRequest endpoint '${
					this.request.url
				}'. This request has been blocked; the content must be served over HTTPS.`,
				DOMExceptionNameEnum.securityError
			);
		}

		const cachedResponse = this.getCachedResponse();

		if (cachedResponse) {
			return cachedResponse;
		}

		if (!this.compliesWithCrossOriginPolicy()) {
			this.#window.console.warn(
				`Cross-Origin Request Blocked: The Same Origin Policy disallows reading the remote resource at "${this.request.url}".`
			);
			throw new this.#window.DOMException(
				`Cross-Origin Request Blocked: The Same Origin Policy disallows reading the remote resource at "${this.request.url}".`,
				DOMExceptionNameEnum.networkError
			);
		}

		return this.sendRequest();
	}

	/**
	 * Returns cached response.
	 *
	 * @returns Response.
	 */
	public getCachedResponse(): ISyncResponse | null {
		if (this.disableCache) {
			return null;
		}

		let cachedResponse = this.#browserFrame.page.context.responseCache.get(this.request);

		if (!cachedResponse || cachedResponse.response.waitingForBody) {
			return null;
		}

		if (cachedResponse.state === CachedResponseStateEnum.stale) {
			const headers = new Headers(cachedResponse.request.headers);

			if (cachedResponse.etag) {
				headers.set('If-None-Match', cachedResponse.etag);
			} else {
				if (!cachedResponse.lastModified) {
					return null;
				}
				headers.set('If-Modified-Since', new Date(cachedResponse.lastModified).toUTCString());
			}

			if (cachedResponse.etag || !cachedResponse.staleWhileRevalidate) {
				const fetch = new SyncFetch({
					browserFrame: this.#browserFrame,
					window: this.#window,
					url: this.request.url,
					init: { headers, method: cachedResponse.request.method },
					disableCache: true,
					disableSameOriginPolicy: true
				});

				const validateResponse = <ISyncResponse>fetch.send();
				const body = validateResponse.status !== 304 ? validateResponse.body : null;

				cachedResponse = this.#browserFrame.page.context.responseCache.add(this.request, {
					...validateResponse,
					body,
					waitingForBody: false
				});

				if (validateResponse.status !== 304) {
					return validateResponse;
				}
			} else {
				const fetch = new Fetch({
					browserFrame: this.#browserFrame,
					window: this.#window,
					url: this.request.url,
					init: { headers, method: cachedResponse.request.method },
					disableCache: true,
					disableSameOriginPolicy: true
				});
				fetch.send().then((response) => {
					response.buffer().then((body: Buffer) => {
						this.#browserFrame.page.context.responseCache.add(this.request, {
							...response,
							body,
							waitingForBody: false
						});
					});
				});
			}
		}

		if (!cachedResponse || cachedResponse.response.waitingForBody) {
			return null;
		}

		return {
			status: cachedResponse.response.status,
			statusText: cachedResponse.response.statusText,
			ok: true,
			url: cachedResponse.response.url,
			// TODO: Do we need to add support for redirected responses to the cache?
			redirected: false,
			headers: cachedResponse.response.headers,
			body: cachedResponse.response.body
		};
	}

	/**
	 * Returns virtual server response.
	 *
	 * @returns Response.
	 */
	private getVirtualServerResponse(): ISyncResponse | null {
		const filePath = VirtualServerUtility.getFilepath(this.#window, this.request.url);

		if (!filePath) {
			return null;
		}

		if (this.request.method !== 'GET') {
			this.#browserFrame?.page?.console.error(
				`${this.request.method} ${this.request.url} 404 (Not Found)`
			);
			return VirtualServerUtility.getNotFoundSyncResponse(this.#window);
		}

		let buffer: Buffer;
		try {
			const stat = FS.statSync(filePath);
			buffer = FS.readFileSync(stat.isDirectory() ? Path.join(filePath, 'index.html') : filePath);
		} catch {
			this.#browserFrame?.page?.console.error(
				`${this.request.method} ${this.request.url} 404 (Not Found)`
			);
			return VirtualServerUtility.getNotFoundSyncResponse(this.#window);
		}

		return {
			status: 200,
			statusText: '',
			ok: true,
			url: this.request.url,
			redirected: false,
			headers: new this.#window.Headers(),
			body: buffer
		};
	}

	/**
	 * Checks if the request complies with the Cross-Origin policy.
	 *
	 * @returns True if it complies with the policy.
	 */
	private compliesWithCrossOriginPolicy(): boolean {
		if (
			this.disableSameOriginPolicy ||
			!FetchCORSUtility.isCORS(this.#window.location.href, this.request[PropertySymbol.url])
		) {
			return true;
		}

		const cachedPreflightResponse = this.#browserFrame.page.context.preflightResponseCache.get(
			this.request
		);

		if (cachedPreflightResponse) {
			if (
				cachedPreflightResponse.allowOrigin !== '*' &&
				cachedPreflightResponse.allowOrigin !== this.#window.location.origin
			) {
				return false;
			}

			if (
				cachedPreflightResponse.allowMethods.length !== 0 &&
				!cachedPreflightResponse.allowMethods.includes(this.request.method)
			) {
				return false;
			}

			return true;
		}

		const requestHeaders = [];

		for (const [header] of this.request.headers) {
			requestHeaders.push(header.toLowerCase());
		}

		const corsHeaders = new Headers({
			'Access-Control-Request-Method': this.request.method,
			Origin: this.#window.location.origin
		});

		if (requestHeaders.length > 0) {
			// This intentionally does not use "combine" (comma + space), as the spec dictates.
			// See https://fetch.spec.whatwg.org/#cors-preflight-fetch for more details.
			// Sorting the headers is not required, but can optimize cache hits.
			corsHeaders.set('Access-Control-Request-Headers', requestHeaders.slice().sort().join(','));
		}

		const fetch = new SyncFetch({
			browserFrame: this.#browserFrame,
			window: this.#window,
			url: this.request.url,
			init: { method: 'OPTIONS' },
			disableCache: true,
			disableSameOriginPolicy: true,
			unfilteredHeaders: corsHeaders
		});

		const response = fetch.send();

		if (!response.ok) {
			return false;
		}

		const allowOrigin = response.headers.get('Access-Control-Allow-Origin');

		if (!allowOrigin) {
			return false;
		}

		if (allowOrigin !== '*' && allowOrigin !== this.#window.location.origin) {
			return false;
		}

		const allowMethods: string[] = [];

		if (response.headers.has('Access-Control-Allow-Methods')) {
			const allowMethodsHeader = response.headers.get('Access-Control-Allow-Methods');
			if (allowMethodsHeader !== '*') {
				for (const method of allowMethodsHeader.split(',')) {
					allowMethods.push(method.trim().toUpperCase());
				}
			}
		}

		if (allowMethods.length !== 0 && !allowMethods.includes(this.request.method)) {
			return false;
		}

		// TODO: Add support for more Access-Control-Allow-* headers.

		return true;
	}

	/**
	 * Sends request.
	 *
	 * @returns Response.
	 */
	public sendRequest(): ISyncResponse {
		if (!this.request[PropertySymbol.bodyBuffer] && this.request.body) {
			throw new this.#window.DOMException(
				`Streams are not supported as request body for synchrounous requests.`,
				DOMExceptionNameEnum.notSupportedError
			);
		}

		const script = SyncFetchScriptBuilder.getScript({
			url: this.request[PropertySymbol.url],
			method: this.request.method,
			headers: FetchRequestHeaderUtility.getRequestHeaders({
				browserFrame: this.#browserFrame,
				window: this.#window,
				request: this.request,
				baseHeaders: this.#unfilteredHeaders
			}),
			body: this.request[PropertySymbol.bodyBuffer]
		});

		// Start the other Node Process, executing this string
		const content = ChildProcess.execFileSync(process.argv[0], ['-e', script], {
			encoding: 'buffer',
			maxBuffer: 1024 * 1024 * 1024 // TODO: Consistent buffer size: 1GB.
		});

		// If content length is 0, then there was an error
		if (!content.length) {
			throw new this.#window.DOMException(
				`Synchronous fetch to "${this.request.url}" failed.`,
				DOMExceptionNameEnum.networkError
			);
		}

		const { error, incomingMessage } = <ISyncHTTPResponse>JSON.parse(content.toString());

		if (error) {
			throw new this.#window.DOMException(
				`Synchronous fetch to "${this.request.url}" failed. Error: ${error}`,
				DOMExceptionNameEnum.networkError
			);
		}

		const headers = FetchResponseHeaderUtility.parseResponseHeaders({
			browserFrame: this.#browserFrame,
			requestURL: this.request[PropertySymbol.url],
			rawHeaders: incomingMessage.rawHeaders
		});

		const response: ISyncResponse = {
			status: incomingMessage.statusCode,
			statusText: incomingMessage.statusMessage,
			ok: incomingMessage.statusCode >= 200 && incomingMessage.statusCode < 300,
			url: this.request.url,
			redirected: this.redirectCount > 0,
			headers,
			body: this.parseIResponseBody({
				headers,
				status: incomingMessage.statusCode,
				body: Buffer.from(incomingMessage.data, 'base64')
			})
		};

		const redirectedResponse = this.handleRedirectResponse(response) || response;

		if (!this.disableCache && !redirectedResponse.redirected) {
			this.#browserFrame.page.context.responseCache.add(this.request, {
				status: <number>redirectedResponse.status,
				statusText: <string>redirectedResponse.statusText,
				url: <string>redirectedResponse.url,
				headers: redirectedResponse.headers,
				body: redirectedResponse.body,
				waitingForBody: false
			});
		}

		const interceptedResponse = this.interceptor?.afterSyncResponse
			? this.interceptor.afterSyncResponse({
					window: this.#window,
					response: redirectedResponse,
					request: this.request
				})
			: undefined;
		const returnResponse =
			typeof interceptedResponse === 'object' ? interceptedResponse : redirectedResponse;
		if (!returnResponse.ok) {
			this.#browserFrame?.page?.console.error(
				`${this.request.method} ${this.request.url} ${returnResponse.status} (${returnResponse.statusText})`
			);
		}
		return returnResponse;
	}

	/**
	 * Parses response body.
	 *
	 * @param options Options.
	 * @param options.headers Headers.
	 * @param options.status Status.
	 * @param options.body Body.
	 * @returns Parsed body.
	 */
	private parseIResponseBody(options: { headers: Headers; status: number; body: Buffer }): Buffer {
		const contentEncodingHeader = options.headers.get('Content-Encoding');

		if (
			this.request.method === 'HEAD' ||
			contentEncodingHeader === null ||
			options.status === 204 ||
			options.status === 304
		) {
			return options.body;
		}

		try {
			// For GZip
			if (contentEncodingHeader === 'gzip' || contentEncodingHeader === 'x-gzip') {
				// Be less strict when decoding compressed responses by using Z_SYNC_FLUSH.
				// Sometimes servers send slightly invalid responses that are still accepted by common browsers.
				// "cURL" always uses Z_SYNC_FLUSH.
				return Zlib.gunzipSync(options.body, {
					flush: Zlib.constants.Z_SYNC_FLUSH,
					finishFlush: Zlib.constants.Z_SYNC_FLUSH
				});
			}

			// For Deflate
			if (contentEncodingHeader === 'deflate' || contentEncodingHeader === 'x-deflate') {
				return Zlib.inflateSync(options.body);
			}

			// For BR
			if (contentEncodingHeader === 'br') {
				return Zlib.brotliDecompressSync(options.body);
			}
		} catch (error) {
			throw new this.#window.DOMException(
				`Failed to read response body. Error: ${error.message}.`,
				DOMExceptionNameEnum.encodingError
			);
		}

		return options.body;
	}

	/**
	 * Handles redirect response.
	 *
	 * @param response Response.
	 * @returns Redirected response or null.
	 */
	private handleRedirectResponse(response: ISyncResponse): ISyncResponse | null {
		if (!FetchResponseRedirectUtility.isRedirect(response.status)) {
			return null;
		}

		switch (this.request.redirect) {
			case 'error':
				throw new this.#window.DOMException(
					`URI requested responds with a redirect, redirect mode is set to "error": ${this.request.url}`,
					DOMExceptionNameEnum.abortError
				);
			case 'manual':
				return null;
			case 'follow':
				const locationHeader = response.headers.get('Location');
				const shouldBecomeGetRequest =
					response.status === 303 ||
					((response.status === 301 || response.status === 302) && this.request.method === 'POST');
				let locationURL: URL = null;

				if (locationHeader !== null) {
					try {
						locationURL = new URL(locationHeader, this.request.url);
					} catch {
						throw new this.#window.DOMException(
							`URI requested responds with an invalid redirect URL: ${locationHeader}`,
							DOMExceptionNameEnum.uriMismatchError
						);
					}
				}

				if (locationURL === null) {
					return null;
				}

				if (FetchResponseRedirectUtility.isMaxRedirectsReached(this.redirectCount)) {
					throw new this.#window.DOMException(
						`Maximum redirects reached at: ${this.request.url}`,
						DOMExceptionNameEnum.networkError
					);
				}

				const headers = new Headers(this.request.headers);
				const requestInit: IRequestInit = {
					method: this.request.method,
					signal: this.request.signal,
					referrer: this.request.referrer,
					referrerPolicy: this.request.referrerPolicy,
					credentials: this.request.credentials,
					headers,
					body: this.request[PropertySymbol.bodyBuffer]
				};

				if (
					this.request.credentials === 'omit' ||
					(this.request.credentials === 'same-origin' &&
						FetchCORSUtility.isCORS(this.#window.location.href, locationURL))
				) {
					headers.delete('authorization');
					headers.delete('www-authenticate');
					headers.delete('cookie');
					headers.delete('cookie2');
				}

				if (shouldBecomeGetRequest) {
					requestInit.method = 'GET';
					requestInit.body = undefined;
					headers.delete('Content-Length');
					headers.delete('Content-Type');
				}

				const responseReferrerPolicy =
					FetchRequestReferrerUtility.getReferrerPolicyFromHeader(headers);
				if (responseReferrerPolicy) {
					requestInit.referrerPolicy = responseReferrerPolicy;
				}

				const fetch = new SyncFetch({
					browserFrame: this.#browserFrame,
					window: this.#window,
					url: locationURL,
					init: requestInit,
					redirectCount: this.redirectCount + 1,
					contentType: !shouldBecomeGetRequest
						? this.request[PropertySymbol.contentType]
						: undefined
				});

				return fetch.send();
			default:
				throw new this.#window.DOMException(
					`Redirect option '${this.request.redirect}' is not a valid value of IRequestRedirect`
				);
		}
	}
}
