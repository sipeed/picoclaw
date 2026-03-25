import Element from '../element/Element.cjs';
import ICachedResult from './ICachedResult.cjs';
import NodeList from './NodeList.cjs';
export default interface ICachedQuerySelectorAllResult extends ICachedResult {
    result: WeakRef<NodeList<Element>> | null;
}
//# sourceMappingURL=ICachedQuerySelectorAllResult.d.ts.map