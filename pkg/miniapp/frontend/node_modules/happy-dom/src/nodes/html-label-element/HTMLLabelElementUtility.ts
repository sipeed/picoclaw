import Document from '../document/Document.js';
import HTMLElement from '../html-element/HTMLElement.js';
import HTMLLabelElement from './HTMLLabelElement.js';
import NodeList from '../node/NodeList.js';
import ShadowRoot from '../shadow-root/ShadowRoot.js';
import * as PropertySymbol from '../../PropertySymbol.js';

/**
 * Utility for finding labels associated with a form element.
 */
export default class HTMLLabelElementUtility {
	/**
	 * Returns label elements for a form element.
	 *
	 * @param element Element to get labels for.
	 * @returns Label elements.
	 */
	public static getAssociatedLabelElements(element: HTMLElement): NodeList<HTMLLabelElement> {
		const id = element.id;
		let labels: HTMLLabelElement[];

		if (id && element[PropertySymbol.isConnected]) {
			const rootNode =
				<Document | ShadowRoot>element[PropertySymbol.rootNode] ||
				element[PropertySymbol.ownerDocument];
			labels = <HTMLLabelElement[]>(
				rootNode.querySelectorAll(`label[for="${id}"]`)[PropertySymbol.items]
			);
		} else {
			labels = [];
		}

		let parent = element[PropertySymbol.parentNode];
		while (parent) {
			if (parent['tagName'] === 'LABEL') {
				labels.push(<HTMLLabelElement>parent);
				break;
			}
			parent = parent[PropertySymbol.parentNode];
		}

		return new NodeList<HTMLLabelElement>(PropertySymbol.illegalConstructor, labels);
	}
}
