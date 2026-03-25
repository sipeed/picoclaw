import Element from '../element/Element.js';
import Node from '../node/Node.js';
export default interface INonDocumentTypeChildNode extends Node {
    readonly previousElementSibling: Element;
    readonly nextElementSibling: Element;
}
//# sourceMappingURL=INonDocumentTypeChildNode.d.ts.map