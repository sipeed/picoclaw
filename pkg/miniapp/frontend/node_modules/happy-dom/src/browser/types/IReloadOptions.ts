/**
 * Reload options.
 */
export default interface IReloadOptions {
	/**
	 * Set to true to bypass the cache.
	 */
	hard?: boolean;

	/**
	 * Timeout in ms. Default is 30000ms.
	 */
	timeout?: number;
}
