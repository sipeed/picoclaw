import Document from '../document/Document.cjs';
import * as PropertySymbol from '../../PropertySymbol.cjs';
import Node from '../node/Node.cjs';
/**
 * Document.
 */
export default class HTMLDocument extends Document {
    /**
     * Constructor.
     */
    constructor();
    /**
     * @override
     */
    [PropertySymbol.appendChild](node: Node, disableValidations?: boolean): Node;
    /**
     * @override
     */
    [PropertySymbol.insertBefore](newNode: Node, referenceNode: Node | null, disableValidations?: boolean): Node;
}
//# sourceMappingURL=HTMLDocument.d.ts.map