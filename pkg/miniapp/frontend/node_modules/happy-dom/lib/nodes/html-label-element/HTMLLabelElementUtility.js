import NodeList from '../node/NodeList.js';
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
    static getAssociatedLabelElements(element) {
        const id = element.id;
        let labels;
        if (id && element[PropertySymbol.isConnected]) {
            const rootNode = element[PropertySymbol.rootNode] ||
                element[PropertySymbol.ownerDocument];
            labels = (rootNode.querySelectorAll(`label[for="${id}"]`)[PropertySymbol.items]);
        }
        else {
            labels = [];
        }
        let parent = element[PropertySymbol.parentNode];
        while (parent) {
            if (parent['tagName'] === 'LABEL') {
                labels.push(parent);
                break;
            }
            parent = parent[PropertySymbol.parentNode];
        }
        return new NodeList(PropertySymbol.illegalConstructor, labels);
    }
}
//# sourceMappingURL=HTMLLabelElementUtility.js.map