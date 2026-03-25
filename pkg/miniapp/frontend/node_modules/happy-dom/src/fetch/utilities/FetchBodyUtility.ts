import MultipartFormDataParser from '../multipart/MultipartFormDataParser.js';
import { ReadableStream } from 'stream/web';
import * as PropertySymbol from '../../PropertySymbol.js';
import { URLSearchParams } from 'url';
import FormData from '../../form-data/FormData.js';
import Blob from '../../file/Blob.js';
import DOMException from '../../exception/DOMException.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import IRequestBody from '../types/IRequestBody.js';
import IResponseBody from '../types/IResponseBody.js';
import { Buffer } from 'buffer';
import Stream from 'stream';
import BrowserWindow from '../../window/BrowserWindow.js';

/**
 * Fetch body utility.
 */
export default class FetchBodyUtility {
	/**
	 * Parses body and returns stream and type.
	 *
	 * Based on:
	 * https://github.com/node-fetch/node-fetch/blob/main/src/body.js (MIT)
	 *
	 * @param body Body.
	 * @returns Stream and type.
	 */
	public static getBodyStream(body: IRequestBody | IResponseBody): {
		contentType: string;
		contentLength: number | null;
		stream: ReadableStream | null;
		buffer: Buffer | null;
	} {
		if (body === null || body === undefined) {
			return { stream: null, buffer: null, contentType: null, contentLength: null };
		} else if (body instanceof URLSearchParams) {
			const buffer = Buffer.from(body.toString());
			return {
				buffer,
				stream: this.toReadableStream(buffer),
				contentType: 'application/x-www-form-urlencoded;charset=UTF-8',
				contentLength: buffer.length
			};
		} else if (body instanceof Blob) {
			const buffer = (<Blob>body)[PropertySymbol.buffer];
			return {
				buffer,
				stream: this.toReadableStream(buffer),
				contentType: body.type,
				contentLength: body.size
			};
		} else if (Buffer.isBuffer(body)) {
			return {
				buffer: body,
				stream: this.toReadableStream(body),
				contentType: null,
				contentLength: body.length
			};
		} else if (body instanceof ArrayBuffer) {
			const buffer = Buffer.from(body);
			return {
				buffer,
				stream: this.toReadableStream(buffer),
				contentType: null,
				contentLength: body.byteLength
			};
		} else if (ArrayBuffer.isView(body)) {
			const buffer = Buffer.from(body.buffer, body.byteOffset, body.byteLength);
			return {
				buffer,
				stream: this.toReadableStream(buffer),
				contentType: null,
				contentLength: body.byteLength
			};
		} else if (body instanceof ReadableStream) {
			return {
				buffer: null,
				stream: body,
				contentType: null,
				contentLength: null
			};
		} else if (body instanceof FormData) {
			return MultipartFormDataParser.formDataToStream(body);
		}

		const buffer = Buffer.from(String(body));
		return {
			buffer,
			stream: this.toReadableStream(buffer),
			contentType: 'text/plain;charset=UTF-8',
			contentLength: buffer.length
		};
	}

	/**
	 * Clones a request or body body stream.
	 *
	 * It is actually not cloning the stream.
	 * It creates a pass through stream and pipes the original stream to it.
	 *
	 * @param window Window.
	 * @param requestOrResponse Request or Response.
	 * @param requestOrResponse.body Body.
	 * @param requestOrResponse.bodyUsed Body used.
	 * @returns New stream.
	 */
	public static cloneBodyStream(
		window: BrowserWindow,
		requestOrResponse: {
			body: ReadableStream | null;
			bodyUsed: boolean;
		}
	): ReadableStream {
		if (requestOrResponse.bodyUsed) {
			throw new window.DOMException(
				`Failed to clone body stream of request: Request body is already used.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		if (requestOrResponse.body === null || requestOrResponse.body === undefined) {
			return null;
		}

		// If a buffer is set, use it to create a new stream.
		if (requestOrResponse[PropertySymbol.buffer]) {
			return this.toReadableStream(requestOrResponse[PropertySymbol.buffer]);
		}

		// Pipe underlying node stream if it exists.
		if (requestOrResponse.body[PropertySymbol.nodeStream]) {
			const stream1 = new Stream.PassThrough();
			const stream2 = new Stream.PassThrough();
			requestOrResponse.body[PropertySymbol.nodeStream].pipe(stream1);
			requestOrResponse.body[PropertySymbol.nodeStream].pipe(stream2);
			// Sets the body of the cloned request/response to the first pass through stream.
			requestOrResponse.body = this.nodeToWebStream(stream1);
			// Returns the clone.
			return this.nodeToWebStream(stream2);
		}

		// Uses the tee() method to clone the ReadableStream
		// This requires the stream to be consumed in parallel which is not the case for the fetch API
		const [stream1, stream2] = requestOrResponse.body.tee();

		// Sets the body of the cloned request to the first pass through stream.
		// TODO: check id this is required as request should be read only object
		<ReadableStream>requestOrResponse.body == stream1;

		// Returns the other stream as the clone
		return stream2;
	}

	/**
	 * Consume and convert an entire Body to a Buffer.
	 *
	 * Based on:
	 * https://github.com/node-fetch/node-fetch/blob/main/src/body.js (MIT)
	 *
	 * @see https://fetch.spec.whatwg.org/#concept-body-consume-body
	 * @param window Window.
	 * @param body Body stream.
	 * @returns Promise.
	 */
	public static async consumeBodyStream(
		window: BrowserWindow,
		body: ReadableStream | null
	): Promise<Buffer> {
		if (body === null || !(body instanceof ReadableStream)) {
			return Buffer.alloc(0);
		}

		if (body[PropertySymbol.error]) {
			throw body[PropertySymbol.error];
		}

		const reader = body.getReader();
		const chunks = [];
		let bytes = 0;

		try {
			let readResult = await reader.read();
			while (!readResult.done) {
				if (body[PropertySymbol.error]) {
					throw body[PropertySymbol.error];
				}
				if (body[PropertySymbol.aborted]) {
					throw new window.DOMException(
						'Failed to read response body: The stream was aborted.',
						DOMExceptionNameEnum.abortError
					);
				}
				const chunk = readResult.value;
				bytes += chunk.length;
				chunks.push(chunk);
				readResult = await reader.read();
			}
		} catch (error) {
			if (error instanceof DOMException) {
				throw error;
			}
			throw new window.DOMException(
				`Failed to read response body. Error: ${error.message}.`,
				DOMExceptionNameEnum.encodingError
			);
		}

		try {
			if (typeof chunks[0] === 'string') {
				return Buffer.from(chunks.join(''));
			}

			return Buffer.concat(chunks, bytes);
		} catch (error) {
			throw new window.DOMException(
				`Could not create Buffer from response body. Error: ${error.message}.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}
	}
	/**
	 * Wraps a given value in a browser ReadableStream.
	 *
	 * This method creates a ReadableStream and immediately enqueues and closes it
	 * with the provided value, useful for stream API compatibility.
	 *
	 * @param value The value to be wrapped in a ReadableStream.
	 * @returns ReadableStream
	 */
	public static toReadableStream(value): ReadableStream {
		return new ReadableStream({
			start(controller) {
				controller.enqueue(value);
				controller.close();
			}
		});
	}

	/**
	 * Wraps a Node.js stream into a browser-compatible ReadableStream.
	 *
	 * Enables the use of Node.js streams where browser ReadableStreams are required.
	 * Handles 'data', 'end', and 'error' events from the Node.js stream.
	 *
	 * @param nodeStream The Node.js stream to be converted.
	 * @returns ReadableStream
	 */
	public static nodeToWebStream(nodeStream: Stream): ReadableStream {
		const readableStream = new ReadableStream({
			start(controller) {
				nodeStream.on('data', (chunk) => {
					controller.enqueue(chunk);
				});

				nodeStream.on('end', () => {
					controller.close();
				});

				nodeStream.on('error', (err) => {
					controller.error(err);
				});
			}
		});
		readableStream[PropertySymbol.nodeStream] = nodeStream;
		return readableStream;
	}
}
