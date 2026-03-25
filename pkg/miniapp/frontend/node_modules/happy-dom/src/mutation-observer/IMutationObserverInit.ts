/**
 * Mutation observer init configurations.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MutationObserverInit
 */
export default interface IMutationObserverInit {
	attributeFilter?: string[];
	attributeOldValue?: boolean;
	attributes?: boolean;
	characterData?: boolean;
	characterDataOldValue?: boolean;
	childList?: boolean;
	subtree?: boolean;
}
