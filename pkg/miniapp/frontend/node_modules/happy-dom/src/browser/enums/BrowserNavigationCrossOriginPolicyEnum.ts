enum BrowserNavigationCrossOriginPolicyEnum {
	/** The browser can navigate to any origin. */
	anyOrigin = 'anyOrigin',
	/** The browser can only navigate to the same origin as the current page or its parent. */
	sameOrigin = 'sameOrigin',
	/** The browser can never navigate from a secure protocol (https) to an unsecure protocol (http), but it can always navigate to a secure (https). */
	strictOrigin = 'strictOrigin'
}

export default BrowserNavigationCrossOriginPolicyEnum;
