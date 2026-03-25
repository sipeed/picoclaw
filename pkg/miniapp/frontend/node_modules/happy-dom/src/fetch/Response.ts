import * as PropertySymbol from '../PropertySymbol.js';
import Blob from '../file/Blob.js';
import IResponseInit from './types/IResponseInit.js';
import IResponseBody from './types/IResponseBody.js';
import Headers from './Headers.js';
import { URLSearchParams } from 'url';
import URL from '../url/URL.js';
import { ReadableStream } from 'stream/web';
import FormData from '../form-data/FormData.js';
import FetchBodyUtility from './utilities/FetchBodyUtility.js';
import DOMExceptionNameEnum from '../exception/DOMExceptionNameEnum.js';
import MultipartFormDataParser from './multipart/MultipartFormDataParser.js';
import BrowserWindow from '../window/BrowserWindow.js';
import ICachedResponse from './cache/response/ICachedResponse.js';
import { Buffer } from 'buffer';
import WindowBrowserContext from '../window/WindowBrowserContext.js';

const REDIRECT_STATUS_CODES = [301, 302, 303, 307, 308];

/**
 * Fetch response.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/response.js (MIT)
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/Response/Response
 */
export default class Response implements Response {
	// Injected by WindowContextClassExtender
	protected declare static [PropertySymbol.window]: BrowserWindow;
	protected declare [PropertySymbol.window]: BrowserWindow;

	// Public properties
	public readonly body: ReadableStream | null = null;
	public readonly bodyUsed = false;
	public readonly redirected = false;
	public readonly type: 'basic' | 'cors' | 'default' | 'error' | 'opaque' | 'opaqueredirect' =
		'basic';
	public readonly url: string = '';
	public readonly status: number;
	public readonly statusText: string;
	public readonly ok: boolean;
	public readonly headers: Headers;
	public [PropertySymbol.cachedResponse]: ICachedResponse | null = null;
	public [PropertySymbol.buffer]: Buffer | null = null;

	/**
	 * Constructor.
	 *
	 * @param body Body.
	 * @param [init] Init.
	 */
	constructor(body?: IResponseBody, init?: IResponseInit) {
		if (!this[PropertySymbol.window]) {
			throw new TypeError(
				`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`
			);
		}

		this.status = init?.status !== undefined ? init.status : 200;
		this.statusText = init?.statusText || '';
		this.ok = this.status >= 200 && this.status < 300;
		this.headers = new Headers(init?.headers);

		// "Set-Cookie" and "Set-Cookie2" are not allowed in response headers according to spec.
		this.headers.delete('Set-Cookie');
		this.headers.delete('Set-Cookie2');

		if (body) {
			const { stream, buffer, contentType } = FetchBodyUtility.getBodyStream(body);
			this.body = stream;

			if (buffer) {
				this[PropertySymbol.buffer] = buffer;
			}

			if (contentType && !this.headers.has('Content-Type')) {
				this.headers.set('Content-Type', contentType);
			}
		}
	}

	/**
	 * Returns string tag.
	 *
	 * @returns String tag.
	 */
	public get [Symbol.toStringTag](): string {
		return 'Response';
	}

	/**
	 * Returns array buffer.
	 *
	 * @returns Array buffer.
	 */
	public async arrayBuffer(): Promise<ArrayBuffer> {
		const window = this[PropertySymbol.window];

		if (this.bodyUsed) {
			throw new window.DOMException(
				`Body has already been used for "${this.url}".`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		const browserFrame = new WindowBrowserContext(window).getBrowserFrame();

		// No browser frame means that the browser is being teared down.
		if (!browserFrame) {
			return new ArrayBuffer(0);
		}

		const asyncTaskManager = browserFrame[PropertySymbol.asyncTaskManager];

		(<boolean>this.bodyUsed) = true;

		let buffer: Buffer | null = this[PropertySymbol.buffer];

		if (!buffer) {
			const taskID = asyncTaskManager.startTask(() => {
				if (this.body) {
					this.body[PropertySymbol.aborted] = true;
				}
			});

			try {
				buffer = await FetchBodyUtility.consumeBodyStream(window, this.body);
			} catch (error) {
				asyncTaskManager.endTask(taskID);
				throw error;
			}

			asyncTaskManager.endTask(taskID);
		}

		this.#storeBodyInCache(buffer);

		return buffer.buffer.slice(buffer.byteOffset, buffer.byteOffset + buffer.byteLength);
	}

	/**
	 * Returns blob.
	 *
	 * @returns Blob.
	 */
	public async blob(): Promise<Blob> {
		const type = this.headers.get('Content-Type') || '';
		const buffer = await this.arrayBuffer();

		return new Blob([buffer], { type });
	}

	/**
	 * Returns buffer.
	 *
	 * @returns Buffer.
	 */
	public async buffer(): Promise<Buffer> {
		const window = this[PropertySymbol.window];

		if (this.bodyUsed) {
			throw new window.DOMException(
				`Body has already been used for "${this.url}".`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		const browserFrame = new WindowBrowserContext(window).getBrowserFrame();

		// No browser frame means that the browser is being teared down.
		if (!browserFrame) {
			return Buffer.alloc(0);
		}

		const asyncTaskManager = browserFrame[PropertySymbol.asyncTaskManager];

		(<boolean>this.bodyUsed) = true;

		let buffer: Buffer | null = this[PropertySymbol.buffer];

		if (!buffer) {
			const taskID = asyncTaskManager.startTask(() => {
				if (this.body) {
					this.body[PropertySymbol.aborted] = true;
				}
			});
			try {
				buffer = await FetchBodyUtility.consumeBodyStream(window, this.body);
			} catch (error) {
				asyncTaskManager.endTask(taskID);
				throw error;
			}
			asyncTaskManager.endTask(taskID);
		}

		this.#storeBodyInCache(buffer);

		return buffer;
	}

	/**
	 * Returns text.
	 *
	 * @returns Text.
	 */
	public async text(): Promise<string> {
		const window = this[PropertySymbol.window];

		if (this.bodyUsed) {
			throw new window.DOMException(
				`Body has already been used for "${this.url}".`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		const browserFrame = new WindowBrowserContext(window).getBrowserFrame();

		// No browser frame means that the browser is being teared down.
		if (!browserFrame) {
			return '';
		}

		const asyncTaskManager = browserFrame[PropertySymbol.asyncTaskManager];

		(<boolean>this.bodyUsed) = true;

		let buffer: Buffer | null = this[PropertySymbol.buffer];

		if (!buffer) {
			const taskID = asyncTaskManager.startTask(() => {
				if (this.body) {
					this.body[PropertySymbol.aborted] = true;
				}
			});
			try {
				buffer = await FetchBodyUtility.consumeBodyStream(window, this.body);
			} catch (error) {
				asyncTaskManager.endTask(taskID);
				throw error;
			}
			asyncTaskManager.endTask(taskID);
		}

		this.#storeBodyInCache(buffer);

		return new TextDecoder().decode(buffer);
	}

	/**
	 * Returns json.
	 *
	 * @returns JSON.
	 */
	public async json(): Promise<string> {
		const text = await this.text();
		return JSON.parse(text);
	}

	/**
	 * Returns form data.
	 *
	 * @returns Form data.
	 */
	public async formData(): Promise<FormData> {
		const window = this[PropertySymbol.window];
		const browserFrame = new WindowBrowserContext(window).getBrowserFrame();

		// No browser frame means that the browser is being teared down.
		if (!browserFrame) {
			return new window.FormData();
		}

		const asyncTaskManager = browserFrame[PropertySymbol.asyncTaskManager];
		const contentType = this.headers.get('Content-Type');

		if (/multipart/i.test(contentType)) {
			if (this.bodyUsed) {
				throw new window.DOMException(
					`Body has already been used for "${this.url}".`,
					DOMExceptionNameEnum.invalidStateError
				);
			}

			(<boolean>this.bodyUsed) = true;

			const taskID = browserFrame[PropertySymbol.asyncTaskManager].startTask(() => {
				if (this.body) {
					this.body[PropertySymbol.aborted] = true;
				}
			});
			let formData: FormData;
			let buffer: Buffer;

			try {
				const result = await MultipartFormDataParser.streamToFormData(
					window,
					this.body,
					contentType
				);
				formData = result.formData;
				buffer = result.buffer;
			} catch (error) {
				asyncTaskManager.endTask(taskID);
				throw error;
			}

			this.#storeBodyInCache(buffer);
			asyncTaskManager.endTask(taskID);

			return formData;
		}

		if (contentType?.startsWith('application/x-www-form-urlencoded')) {
			const parameters = new URLSearchParams(await this.text());
			const formData = new window.FormData();

			for (const [key, value] of parameters) {
				formData.append(key, value);
			}

			return formData;
		}

		throw new window.DOMException(
			`Failed to build FormData object: The "content-type" header is neither "application/x-www-form-urlencoded" nor "multipart/form-data".`,
			DOMExceptionNameEnum.invalidStateError
		);
	}

	/**
	 * Clones request.
	 *
	 * @returns Clone.
	 */
	public clone(): Response {
		const window = this[PropertySymbol.window];
		const body = FetchBodyUtility.cloneBodyStream(window, this);

		const response = new window.Response(body, {
			status: this.status,
			statusText: this.statusText,
			headers: this.headers
		});

		response[PropertySymbol.cachedResponse] = this[PropertySymbol.cachedResponse];
		response[PropertySymbol.buffer] = this[PropertySymbol.buffer];
		(<boolean>response.ok) = this.ok;
		(<boolean>response.redirected) = this.redirected;
		(<string>response.type) = this.type;
		(<string>response.url) = this.url;

		return response;
	}

	/**
	 * Stores body in cache.
	 *
	 * @param buffer Buffer.
	 */
	#storeBodyInCache(buffer: Buffer): void {
		if (this[PropertySymbol.cachedResponse]?.response?.waitingForBody) {
			this[PropertySymbol.cachedResponse].response.body = buffer;
			this[PropertySymbol.cachedResponse].response.waitingForBody = false;
		}
	}

	/**
	 * Returns a redirect response.
	 *
	 * @param url URL.
	 * @param status Status code.
	 * @returns Response.
	 */
	public static redirect(url: string, status = 302): Response {
		const window = this[PropertySymbol.window];

		if (!REDIRECT_STATUS_CODES.includes(status)) {
			throw new window.DOMException(
				'Failed to create redirect response: Invalid redirect status code.',
				DOMExceptionNameEnum.invalidStateError
			);
		}

		return new window.Response(null, {
			headers: {
				location: new URL(url).toString()
			},
			status
		});
	}

	/**
	 * Returns an error response.
	 *
	 * @param url URL.
	 * @param status Status code.
	 * @returns Response.
	 */
	public static error(): Response {
		const response = new this[PropertySymbol.window].Response(null, { status: 0, statusText: '' });
		(<string>response.type) = 'error';
		return response;
	}

	/**
	 * Returns an JSON response.
	 *
	 * @param injected Injected properties.
	 * @param data Data.
	 * @param [init] Init.
	 * @returns Response.
	 */
	public static json(data: object, init?: IResponseInit): Response {
		const window = this[PropertySymbol.window];
		const body = JSON.stringify(data);

		if (body === undefined) {
			throw new window.TypeError('data is not JSON serializable');
		}

		const headers = new window.Headers(init && init.headers);

		if (!headers.has('Content-Type')) {
			headers.set('Content-Type', 'application/json');
		}

		return new window.Response(body, {
			status: 200,
			...init,
			headers
		});
	}
}
