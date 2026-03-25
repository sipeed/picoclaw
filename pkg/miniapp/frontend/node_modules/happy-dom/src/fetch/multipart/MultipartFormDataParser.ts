import FormData from '../../form-data/FormData.js';
import { ReadableStream } from 'stream/web';
import * as PropertySymbol from '../../PropertySymbol.js';
import MultipartReader from './MultipartReader.js';
import DOMExceptionNameEnum from '../../exception/DOMExceptionNameEnum.js';
import { Buffer } from 'buffer';
import BrowserWindow from '../../window/BrowserWindow.js';

/**
 * Multipart form data factory.
 *
 * Based on:
 * https://github.com/node-fetch/node-fetch/blob/main/src/utils/multipart-parser.js (MIT)
 */
export default class MultipartFormDataParser {
	/**
	 * Returns form data.
	 *
	 * @param window Window.
	 * @param body Body.
	 * @param contentType Content type header value.
	 * @returns Form data.
	 */
	public static async streamToFormData(
		window: BrowserWindow,
		body: ReadableStream,
		contentType: string
	): Promise<{ formData: FormData; buffer: Buffer }> {
		if (!/multipart/i.test(contentType)) {
			throw new window.DOMException(
				`Failed to build FormData object: The "content-type" header isn't of type "multipart/form-data".`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		const match = contentType.match(/boundary=(?:"([^"]+)"|([^;]+))/i);

		if (!match) {
			throw new window.DOMException(
				`Failed to build FormData object: The "content-type" header doesn't contain any multipart boundary.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		const bodyReader = body.getReader();
		const reader = new MultipartReader(window, match[1] || match[2]);
		const chunks = [];
		let buffer: Buffer;
		const bytes = 0;

		let readResult = await bodyReader.read();

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
			reader.write(readResult.value);
			readResult = await bodyReader.read();
		}

		try {
			buffer =
				typeof chunks[0] === 'string' ? Buffer.from(chunks.join('')) : Buffer.concat(chunks, bytes);
		} catch (error) {
			throw new window.DOMException(
				`Could not create Buffer from response body. Error: ${error.message}.`,
				DOMExceptionNameEnum.invalidStateError
			);
		}

		return {
			formData: reader.end(),
			buffer
		};
	}

	/**
	 * Converts a FormData object to a ReadableStream.
	 *
	 * @param formData FormData.
	 * @returns Stream and type.
	 */
	public static formDataToStream(formData: FormData): {
		contentType: string;
		contentLength: number;
		buffer: Buffer;
		stream: ReadableStream;
	} {
		const boundary = '----HappyDOMFormDataBoundary' + Math.random().toString(36);
		const chunks: Buffer[] = [];
		const prefix = `--${boundary}\r\nContent-Disposition: form-data; name="`;

		for (const [name, value] of formData) {
			if (typeof value === 'string') {
				chunks.push(
					Buffer.from(
						`${prefix}${this.escapeName(name)}"\r\n\r\n${value.replace(
							/\r(?!\n)|(?<!\r)\n/g,
							'\r\n'
						)}\r\n`
					)
				);
			} else {
				chunks.push(
					Buffer.from(
						`${prefix}${this.escapeName(name)}"; filename="${this.escapeName(
							value.name,
							true
						)}"\r\nContent-Type: ${value.type || 'application/octet-stream'}\r\n\r\n`
					)
				);
				chunks.push(value[PropertySymbol.buffer]);
				chunks.push(Buffer.from('\r\n'));
			}
		}

		const buffer = Buffer.concat(chunks);

		return {
			contentType: `multipart/form-data; boundary=${boundary}`,
			contentLength: buffer.length,
			buffer,
			stream: new ReadableStream({
				start(controller) {
					controller.enqueue(buffer);
					controller.close();
				}
			})
		};
	}

	/**
	 * Escapes a form data entry name.
	 *
	 * @param name Name.
	 * @param filename Whether it is a filename.
	 * @returns Escaped name.
	 */
	private static escapeName(name: string, filename = false): string {
		return (filename ? name : name.replace(/\r?\n|\r/g, '\r\n'))
			.replace(/\n/g, '%0A')
			.replace(/\r/g, '%0D')
			.replace(/"/g, '%22');
	}
}
