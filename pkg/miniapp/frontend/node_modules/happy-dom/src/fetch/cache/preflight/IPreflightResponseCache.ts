import ICachedPreflightResponse from './ICachedPreflightResponse.js';
import ICachablePreflightRequest from './ICachablePreflightRequest.js';
import ICachablePreflightResponse from './ICachablePreflightResponse.js';

/**
 * Fetch response cache.
 */
export default interface IPreflightResponseCache {
	/**
	 * Returns cached response.
	 *
	 * @param request Request.
	 * @returns Cached response.
	 */
	get(request: ICachablePreflightRequest): ICachedPreflightResponse | null;

	/**
	 * Adds a cached response.
	 *
	 * @param request Request.
	 * @param response Response.
	 * @returns Cached response.
	 */
	add(
		request: ICachablePreflightRequest,
		response: ICachablePreflightResponse
	): ICachedPreflightResponse | null;

	/**
	 * Clears the cache.
	 */
	clear(): void;
}
