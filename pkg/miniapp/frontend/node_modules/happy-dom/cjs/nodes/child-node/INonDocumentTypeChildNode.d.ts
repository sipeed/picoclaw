import Element from '../element/Element.cjs';
import Node from '../node/Node.cjs';
export default interface INonDocumentTypeChildNode extends Node {
    readonly previousElementSibling: Element;
    readonly nextElementSibling: Element;
}
//# sourceMappingURL=INonDocumentTypeChildNode.d.ts.map