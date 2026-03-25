import Element from '../nodes/element/Element.cjs';
import NodeList from '../nodes/node/NodeList.cjs';
import Document from '../nodes/document/Document.cjs';
import DocumentFragment from '../nodes/document-fragment/DocumentFragment.cjs';
import ISelectorMatch from './ISelectorMatch.cjs';
import IHTMLElementTagNameMap from '../config/IHTMLElementTagNameMap.cjs';
import ISVGElementTagNameMap from '../config/ISVGElementTagNameMap.cjs';
/**
 * Utility for query selection in an HTML element.
 *
 * @class QuerySelector
 */
export default class QuerySelector {
    /**
     * Finds elements based on a query selector.
     *
     * @param node Node to search in.
     * @param selector Selector.
     * @returns HTML elements.
     */
    static querySelectorAll<K extends keyof IHTMLElementTagNameMap>(node: Element | Document | DocumentFragment, selector: K): NodeList<IHTMLElementTagNameMap[K]>;
    /**
     * Finds elements based on a query selector.
     *
     * @param node Node to search in.
     * @param selector Selector.
     * @returns HTML elements.
     */
    static querySelectorAll<K extends keyof ISVGElementTagNameMap>(node: Element | Document | DocumentFragment, selector: K): NodeList<ISVGElementTagNameMap[K]>;
    /**
     * Finds elements based on a query selector.
     *
     * @param node Node to search in.
     * @param selector Selector.
     * @returns HTML elements.
     */
    static querySelectorAll(node: Element | Document | DocumentFragment, selector: string): NodeList<Element>;
    /**
     * Finds an element based on a query selector.
     *
     * @param node Node to search in.
     * @param selector Selector.
     * @returns HTML element.
     */
    static querySelector<K extends keyof IHTMLElementTagNameMap>(node: Element | Document | DocumentFragment, selector: K): IHTMLElementTagNameMap[K] | null;
    /**
     * Finds an element based on a query selector.
     *
     * @param node Node to search in.
     * @param selector Selector.
     * @returns HTML element.
     */
    static querySelector<K extends keyof ISVGElementTagNameMap>(node: Element | Document | DocumentFragment, selector: K): ISVGElementTagNameMap[K] | null;
    /**
     * Finds an element based on a query selector.
     *
     * @param node Node to search in.
     * @param selector Selector.
     * @returns HTML element.
     */
    static querySelector(node: Element | Document | DocumentFragment, selector: string): Element | null;
    /**
     * Checks if an element matches a selector and returns priority weight.
     *
     * @param element Element to match.
     * @param selector Selector to match with.
     * @param [options] Options.
     * @param [options.ignoreErrors] Ignores errors.
     * @returns Result.
     */
    static matches(element: Element, selector: string, options?: {
        ignoreErrors?: boolean;
    }): ISelectorMatch | null;
    /**
     * Checks if a node matches a selector.
     *
     * @param element Target element.
     * @param currentElement
     * @param selectorItems Selector items.
     * @param cachedItem Cached item.
     * @param [previousSelectorItem] Previous selector item.
     * @param [priorityWeight] Priority weight.
     * @returns Result.
     */
    private static matchSelector;
    /**
     * Finds elements based on a query selector for a part of a list of selectors separated with comma.
     *
     * @param rootElement Root element.
     * @param children Child elements.
     * @param selectorItems Selector items.
     * @param cachedItem Cached item.
     * @param [documentPosition] Document position of the element.
     * @returns Document position and element map.
     */
    private static findAll;
    /**
     * Finds an element based on a query selector for a part of a list of selectors separated with comma.
     *
     * @param rootElement Root element.
     * @param children Child elements.
     * @param selectorItems Selector items.
     * @param cachedItem Cached item.
     * @returns HTML element.
     */
    private static findFirst;
}
//# sourceMappingURL=QuerySelector.d.ts.map