import IResponseCache from './IResponseCache.js';
import ICachedResponse from './ICachedResponse.js';
import CachedResponseStateEnum from './CachedResponseStateEnum.js';
import ICachableRequest from './ICachableRequest.js';
import ICachableResponse from './ICachableResponse.js';
import Headers from '../../Headers.js';

const UPDATE_RESPONSE_HEADERS = ['Cache-Control', 'Last-Modified', 'Vary', 'ETag'];

/**
 * Fetch response cache.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/HTTP/Caching
 * @see https://www.mnot.net/cache_docs/
 */
export default class ResponseCache implements IResponseCache {
	#entries: { [url: string]: ICachedResponse[] } = {};

	/**
	 * Returns cached response.
	 *
	 * @param request Request.
	 * @returns Cached response.
	 */
	public get(request: ICachableRequest): ICachedResponse | null {
		if (request.headers.get('Cache-Control')?.includes('no-cache')) {
			return null;
		}

		const url = request.url;

		if (this.#entries[url]) {
			for (let i = 0, max = this.#entries[url].length; i < max; i++) {
				const entry = this.#entries[url][i];
				let isMatch = entry.request.method === request.method;
				if (isMatch) {
					for (const header of Object.keys(entry.vary)) {
						const requestHeader = request.headers.get(header);
						if (requestHeader !== null && entry.vary[header] !== requestHeader) {
							isMatch = false;
							break;
						}
					}
				}
				if (isMatch) {
					if (entry.expires && entry.expires < Date.now()) {
						if (entry.lastModified) {
							entry.state = CachedResponseStateEnum.stale;
						} else if (!entry.etag) {
							this.#entries[url].splice(i, 1);
							return null;
						}
					}
					return entry;
				}
			}
		}
		return null;
	}

	/**
	 * Adds a cache entity.
	 *
	 * @param request Request.
	 * @param response Response.
	 * @returns Cached response.
	 */
	public add(request: ICachableRequest, response: ICachableResponse): ICachedResponse {
		// We should only cache GET and HEAD requests.
		if (
			(request.method !== 'GET' && request.method !== 'HEAD') ||
			request.headers.get('Cache-Control')?.includes('no-cache')
		) {
			return null;
		}

		const url = request.url;
		let cachedResponse = this.get(request);

		if (response.status === 304) {
			if (!cachedResponse) {
				throw new Error('ResponseCache: Cached response not found.');
			}

			for (const name of UPDATE_RESPONSE_HEADERS) {
				if (response.headers.has(name)) {
					cachedResponse.response.headers.set(name, response.headers.get(name));
				}
			}

			cachedResponse.cacheUpdateTime = Date.now();
			cachedResponse.state = CachedResponseStateEnum.fresh;
		} else {
			if (cachedResponse) {
				const index = this.#entries[url].indexOf(cachedResponse);
				if (index !== -1) {
					this.#entries[url].splice(index, 1);
				}
			}

			cachedResponse = {
				response: {
					status: response.status,
					statusText: response.statusText,
					url: response.url,
					headers: new Headers(response.headers),
					// We need to wait for the body to be consumed and then populated if set to true (e.g. by using Response.text()).
					waitingForBody: response.waitingForBody,
					body: response.body ?? null
				},
				request: {
					headers: request.headers,
					method: request.method
				},
				vary: {},
				expires: null,
				etag: null,
				cacheUpdateTime: Date.now(),
				lastModified: null,
				mustRevalidate: false,
				staleWhileRevalidate: false,
				state: CachedResponseStateEnum.fresh
			};

			this.#entries[url] = this.#entries[url] || [];
			this.#entries[url].push(cachedResponse);
		}

		if (response.headers.has('Cache-Control')) {
			const age = response.headers.get('Age');

			for (const part of response.headers.get('Cache-Control').split(',')) {
				const [key, value] = part.trim().split('=');
				switch (key) {
					case 'max-age':
						cachedResponse.expires =
							Date.now() + parseInt(value) * 1000 - (age ? parseInt(age) * 1000 : 0);
						break;
					case 'no-cache':
					case 'no-store':
						const index = this.#entries[url].indexOf(cachedResponse);
						if (index !== -1) {
							this.#entries[url].splice(index, 1);
						}
						return null;
					case 'must-revalidate':
						cachedResponse.mustRevalidate = true;
						break;
					case 'stale-while-revalidate':
						cachedResponse.staleWhileRevalidate = true;
						break;
				}
			}
		}

		if (response.headers.has('Last-Modified')) {
			cachedResponse.lastModified = Date.parse(response.headers.get('Last-Modified'));
		}

		if (response.headers.has('Vary')) {
			for (const header of response.headers.get('Vary').split(',')) {
				const name = header.trim();
				const value = request.headers.get(name);
				if (value) {
					cachedResponse.vary[name] = value;
				}
			}
		}

		if (response.headers.has('ETag')) {
			cachedResponse.etag = response.headers.get('ETag');
		}

		if (!cachedResponse.expires) {
			const expires = response.headers.get('Expires');
			if (expires) {
				cachedResponse.expires = Date.parse(expires);
			}
		}

		// Cache is invalid if it has expired and doesn't have an ETag.
		if (!cachedResponse.etag && (!cachedResponse.expires || cachedResponse.expires < Date.now())) {
			const index = this.#entries[url].indexOf(cachedResponse);
			if (index !== -1) {
				this.#entries[url].splice(index, 1);
			}
			return null;
		}

		return cachedResponse;
	}

	/**
	 * Clears the cache.
	 *
	 * @param [options] Options.
	 * @param [options.url] URL.
	 * @param [options.toTime] Removes all entries that are older than this time. Time in MS.
	 */
	public clear(options?: { url?: string; toTime?: number }): void {
		if (options) {
			if (options.toTime) {
				for (const key of options.url ? [options.url] : Object.keys(this.#entries)) {
					if (this.#entries[key]) {
						for (let i = 0, max = this.#entries[key].length; i < max; i++) {
							if (this.#entries[key][i].cacheUpdateTime < options.toTime) {
								this.#entries[key].splice(i, 1);
								i--;
								max--;
							}
						}
					}
				}
			} else if (options.url) {
				delete this.#entries[options.url];
			}
		} else {
			this.#entries = {};
		}
	}
}
