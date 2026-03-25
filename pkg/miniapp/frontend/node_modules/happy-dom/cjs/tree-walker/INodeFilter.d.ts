import Node from '../nodes/node/Node.cjs';
type INodeFilter = ((node: Node) => number) | {
    acceptNode(node: Node): number;
};
export default INodeFilter;
//# sourceMappingURL=INodeFilter.d.ts.map