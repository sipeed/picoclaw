import Node from '../nodes/node/Node.js';

type INodeFilter = ((node: Node) => number) | { acceptNode(node: Node): number };

export default INodeFilter;
