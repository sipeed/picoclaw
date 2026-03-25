import XMLHttpRequestEventTarget from './XMLHttpRequestEventTarget.js';
import * as PropertySymbol from '../PropertySymbol.js';
import XMLHttpRequestReadyStateEnum from './XMLHttpRequestReadyStateEnum.js';
import Event from '../event/Event.js';
import Document from '../nodes/document/Document.js';
import Blob from '../file/Blob.js';
import XMLHttpRequestUpload from './XMLHttpRequestUpload.js';
import DOMException from '../exception/DOMException.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import XMLHttpResponseTypeEnum from './XMLHttpResponseTypeEnum.js';
import ErrorEvent from '../event/events/ErrorEvent.js';
import Headers from '../fetch/Headers.js';
import Fetch from '../fetch/Fetch.js';
import SyncFetch from '../fetch/SyncFetch.js';
import Request from '../fetch/Request.js';
import ISyncResponse from '../fetch/types/ISyncResponse.js';
import AbortController from '../fetch/AbortController.js';
import ProgressEvent from '../event/events/ProgressEvent.js';
import NodeTypeEnum from '../nodes/node/NodeTypeEnum.js';
import IRequestBody from '../fetch/types/IRequestBody.js';
import XMLHttpRequestResponseDataParser from './XMLHttpRequestResponseDataParser.js';
import FetchRequestHeaderUtility from '../fetch/utilities/FetchRequestHeaderUtility.js';
import Response from '../fetch/Response.js';
import WindowBrowserContext from '../window/WindowBrowserContext.js';

/**
 * XMLHttpRequest.
 *
 * Based on:
 * https://github.com/mjwwit/node-XMLHttpRequest/blob/master/lib/XMLHttpRequest.js
 */
export default class XMLHttpRequest extends XMLHttpRequestEventTarget {
	// Constants
	public static UNSENT = XMLHttpRequestReadyStateEnum.unsent;
	public static OPENED = XMLHttpRequestReadyStateEnum.opened;
	public static HEADERS_RECEIVED = XMLHttpRequestReadyStateEnum.headersRecieved;
	public static LOADING = XMLHttpRequestReadyStateEnum.loading;
	public static DONE = XMLHttpRequestReadyStateEnum.done;

	// Public properties
	public upload: XMLHttpRequestUpload = new this[PropertySymbol.window].XMLHttpRequestUpload();
	public withCredentials = false;

	// Private properties
	#async = true;
	#abortController: AbortController | null = null;
	#aborted = false;
	#request: Request | null = null;
	#response: Response | ISyncResponse | null = null;
	#responseType: XMLHttpResponseTypeEnum | '' = '';
	#responseBody: ArrayBuffer | Blob | Document | object | string | null = null;
	#readyState: XMLHttpRequestReadyStateEnum = XMLHttpRequestReadyStateEnum.unsent;

	/**
	 * Constructor.
	 */
	constructor() {
		super();

		if (!this[PropertySymbol.window]) {
			throw new TypeError(
				`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`
			);
		}
	}

	/**
	 * Returns the status.
	 *
	 * @returns Status.
	 */
	public get status(): number {
		return this.#response?.status || 0;
	}

	/**
	 * Returns the status text.
	 *
	 * @returns Status text.
	 */
	public get statusText(): string {
		return this.#response?.statusText || '';
	}

	/**
	 * Returns the response.
	 *
	 * @returns Response.
	 */
	public get response(): ArrayBuffer | Blob | Document | object | string | null {
		if (!this.#response) {
			return '';
		}

		return this.#responseBody;
	}

	/**
	 * Get the response text.
	 *
	 * @throws {DOMException} If the response type is not text or empty.
	 * @returns The response text.
	 */
	public get responseText(): string {
		if (this.responseType !== XMLHttpResponseTypeEnum.text && this.responseType !== '') {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to read the 'responseText' property from 'XMLHttpRequest': The value is only accessible if the object's 'responseType' is '' or 'text' (was '${this.responseType}').`,
				DOMExceptionNameEnum.invalidStateError
			);
		}
		return <string>this.#responseBody ?? '';
	}

	/**
	 * Get the responseXML.
	 *
	 * @throws {DOMException} If the response type is not text or empty.
	 * @returns Response XML.
	 */
	public get responseXML(): Document {
		if (this.responseType !== XMLHttpResponseTypeEnum.document && this.responseType !== '') {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to read the 'responseXML' property from 'XMLHttpRequest': The value is only accessible if the object's 'responseType' is '' or 'document' (was '${this.responseType}').`,
				DOMExceptionNameEnum.invalidStateError
			);
		}
		return this.responseType === '' ? null : <Document>this.#responseBody;
	}

	/**
	 * Returns the response URL.
	 *
	 * @returns Response URL.
	 */
	public get responseURL(): string {
		return this.#response?.url || '';
	}

	/**
	 * Returns the ready state.
	 *
	 * @returns Ready state.
	 */
	public get readyState(): XMLHttpRequestReadyStateEnum {
		return this.#readyState;
	}

	/**
	 * Set response type.
	 *
	 * @param type Response type.
	 * @throws {DOMException} If the state is not unsent or opened.
	 * @throws {DOMException} If the request is synchronous.
	 */
	public set responseType(type: XMLHttpResponseTypeEnum | '') {
		// ResponseType can only be set when the state is unsent or opened.
		if (
			this.readyState !== XMLHttpRequestReadyStateEnum.opened &&
			this.readyState !== XMLHttpRequestReadyStateEnum.unsent
		) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to set the 'responseType' property on 'XMLHttpRequest': The object's state must be OPENED or UNSENT.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}
		// Sync requests can only have empty string or 'text' as response type.
		if (!this.#async) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to set the 'responseType' property on 'XMLHttpRequest': The response type cannot be changed for synchronous requests made from a document.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}
		this.#responseType = type;
	}

	/**
	 * Get response Type.
	 *
	 * @returns Response type.
	 */
	public get responseType(): XMLHttpResponseTypeEnum | '' {
		return this.#responseType;
	}

	/**
	 * Opens the connection.
	 *
	 * @param method Connection method (eg GET, POST).
	 * @param url URL for the connection.
	 * @param [async=true] Asynchronous connection.
	 * @param [user] Username for basic authentication (optional).
	 * @param [password] Password for basic authentication (optional).
	 */
	public open(method: string, url: string, async = true, user?: string, password?: string): void {
		const window = this[PropertySymbol.window];

		if (!async && !!this.responseType && this.responseType !== XMLHttpResponseTypeEnum.text) {
			throw new window.DOMException(
				`Failed to execute 'open' on 'XMLHttpRequest': Synchronous requests from a document must not set a response type.`,
				DOMExceptionNameEnum.invalidAccessError
			);
		}

		const headers = new Headers();
		if (user) {
			const authBuffer = Buffer.from(`${user}:${password || ''}`);
			headers.set('Authorization', 'Basic ' + authBuffer.toString('base64'));
		}

		this.#async = async;
		this.#aborted = false;
		this.#response = null;
		this.#responseBody = null;
		this.#abortController = new window.AbortController();
		this.#request = new window.Request(url, {
			method,
			headers,
			signal: this.#abortController.signal,
			credentials: this.withCredentials ? 'include' : 'same-origin'
		});

		this.#readyState = XMLHttpRequestReadyStateEnum.opened;
	}

	/**
	 * Sets a header for the request.
	 *
	 * @param name Header name.
	 * @param value Header value.
	 * @returns Header added.
	 */
	public setRequestHeader(name: string, value: string): boolean {
		if (this.readyState !== XMLHttpRequestReadyStateEnum.opened) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'setRequestHeader' on 'XMLHttpRequest': The object's state must be OPENED.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		// TODO: Use FetchRequestHeaderUtility.removeForbiddenHeaders() instead.
		if (FetchRequestHeaderUtility.isHeaderForbidden(name)) {
			return false;
		}

		this.#request.headers.set(name, value);

		return true;
	}

	/**
	 * Gets a header from the server response.
	 *
	 * @param header header Name of header to get.
	 * @returns string Text of the header or null if it doesn't exist.
	 */
	public getResponseHeader(header: string): string | null {
		return this.#response?.headers.get(header) ?? null;
	}

	/**
	 * Gets all the response headers.
	 *
	 * @returns A string with all response headers separated by CR+LF.
	 */
	public getAllResponseHeaders(): string {
		if (!this.#response) {
			return '';
		}

		const result = [];

		for (const [name, value] of this.#response?.headers) {
			const lowerName = name.toLowerCase();
			if (lowerName !== 'set-cookie' && lowerName !== 'set-cookie2') {
				result.push(`${name}: ${value}`);
			}
		}

		return result.join('\r\n');
	}

	/**
	 * Sends the request to the server.
	 *
	 * @param body Optional data to send as request body.
	 */
	public send(body?: Document | IRequestBody): void {
		const window = this[PropertySymbol.window];

		if (this.readyState != XMLHttpRequestReadyStateEnum.opened) {
			throw new this[PropertySymbol.window].DOMException(
				`Failed to execute 'send' on 'XMLHttpRequest': Connection must be opened before send() is called.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		// When body is a Document, serialize it to a string.
		if (
			typeof body === 'object' &&
			body !== null &&
			(<Document>body)[PropertySymbol.nodeType] === NodeTypeEnum.documentNode
		) {
			body = new window.XMLSerializer().serializeToString(<Document>body);
		}

		if (this.#async) {
			this.#sendAsync(<IRequestBody>body).catch((error) => {
				throw error;
			});
		} else {
			this.#sendSync(<IRequestBody>body);
		}
	}

	/**
	 * Aborts a request.
	 */
	public abort(): void {
		if (this.#aborted) {
			return;
		}
		this.#aborted = true;
		this.#abortController.abort();
	}

	/**
	 * Sends the request to the server asynchronously.
	 *
	 * @param body Optional data to send as request body.
	 */
	async #sendAsync(body?: IRequestBody): Promise<void> {
		const window = this[PropertySymbol.window];
		const browserFrame = new WindowBrowserContext(window).getBrowserFrame();
		const asyncTaskManager = browserFrame[PropertySymbol.asyncTaskManager];
		const taskID = asyncTaskManager.startTask(() => this.abort());

		this.#readyState = XMLHttpRequestReadyStateEnum.loading;

		this.dispatchEvent(new Event('readystatechange'));
		this.dispatchEvent(new Event('loadstart'));

		if (body) {
			this.#request = new window.Request(this.#request.url, {
				method: this.#request.method,
				headers: this.#request.headers,
				signal: this.#abortController.signal,
				credentials: this.#request.credentials,
				body
			});
		}

		this.#abortController.signal.addEventListener('abort', () => {
			this.#aborted = true;
			this.#readyState = XMLHttpRequestReadyStateEnum.unsent;
			this.dispatchEvent(new Event('abort'));
			this.dispatchEvent(new Event('loadend'));
			this.dispatchEvent(new Event('readystatechange'));
			asyncTaskManager.endTask(taskID);
		});

		const onError = (error: Error): void => {
			if (error instanceof DOMException && error.name === DOMExceptionNameEnum.abortError) {
				if (this.#aborted) {
					return;
				}
				this.#readyState = XMLHttpRequestReadyStateEnum.unsent;
				this.dispatchEvent(new Event('abort'));
			} else {
				this.#readyState = XMLHttpRequestReadyStateEnum.done;
				this.dispatchEvent(new ErrorEvent('error', { error, message: error.message }));
			}
			this.dispatchEvent(new Event('loadend'));
			this.dispatchEvent(new Event('readystatechange'));
			asyncTaskManager.endTask(taskID);
		};

		const fetch = new Fetch({
			browserFrame: browserFrame,
			window: window,
			url: this.#request.url,
			init: this.#request
		});

		try {
			this.#response = await fetch.send();
		} catch (error) {
			onError(error);
			return;
		}

		this.#readyState = XMLHttpRequestReadyStateEnum.headersRecieved;

		this.dispatchEvent(new Event('readystatechange'));

		const contentLength = this.#response.headers.get('Content-Length');
		const contentLengthNumber =
			contentLength !== null && !isNaN(Number(contentLength)) ? Number(contentLength) : null;
		let loaded = 0;
		let data = Buffer.from([]);

		if (this.#response.body) {
			let eventError: Error;
			try {
				for await (const chunk of this.#response.body) {
					data = Buffer.concat([data, typeof chunk === 'string' ? Buffer.from(chunk) : chunk]);
					loaded += chunk.length;
					// We need to re-throw the error as we don't want it to be caught by the try/catch.
					try {
						this.dispatchEvent(
							new ProgressEvent('progress', {
								lengthComputable: contentLengthNumber !== null,
								loaded,
								total: contentLengthNumber !== null ? contentLengthNumber : 0
							})
						);
					} catch (error) {
						eventError = error;
						throw error;
					}
				}
			} catch (error) {
				if (error === eventError) {
					throw error;
				}
				onError(error);
				return;
			}
		}

		this.#responseBody = XMLHttpRequestResponseDataParser.parse({
			window: window,
			responseType: this.#responseType,
			data,
			contentType:
				this.#response.headers.get('Content-Type') || this.#request.headers.get('Content-Type')
		});
		this.#readyState = XMLHttpRequestReadyStateEnum.done;

		asyncTaskManager.endTask(taskID);

		this.dispatchEvent(new Event('readystatechange'));
		this.dispatchEvent(new Event('load'));
		this.dispatchEvent(new Event('loadend'));
	}

	/**
	 * Sends the request to the server synchronously.
	 *
	 * @param body Optional data to send as request body.
	 */
	#sendSync(body?: IRequestBody): void {
		const window = this[PropertySymbol.window];
		const browserFrame = new WindowBrowserContext(window).getBrowserFrame();

		if (body) {
			this.#request = new window.Request(this.#request.url, {
				method: this.#request.method,
				headers: this.#request.headers,
				signal: this.#abortController.signal,
				credentials: this.#request.credentials,
				body
			});
		}

		this.#readyState = XMLHttpRequestReadyStateEnum.loading;

		const fetch = new SyncFetch({
			browserFrame,
			window: window,
			url: this.#request.url,
			init: this.#request
		});

		try {
			this.#response = fetch.send();
		} catch (error) {
			this.#readyState = XMLHttpRequestReadyStateEnum.done;
			this.dispatchEvent(new ErrorEvent('error', { error, message: error.message }));
			this.dispatchEvent(new Event('loadend'));
			this.dispatchEvent(new Event('readystatechange'));
			return;
		}

		this.#readyState = XMLHttpRequestReadyStateEnum.headersRecieved;

		this.#responseBody = XMLHttpRequestResponseDataParser.parse({
			window: window,
			responseType: this.#responseType,
			data: this.#response.body,
			contentType:
				this.#response.headers.get('Content-Type') || this.#request.headers.get('Content-Type')
		});

		this.#readyState = XMLHttpRequestReadyStateEnum.done;

		this.dispatchEvent(new Event('readystatechange'));
		this.dispatchEvent(new Event('load'));
		this.dispatchEvent(new Event('loadend'));
	}
}
