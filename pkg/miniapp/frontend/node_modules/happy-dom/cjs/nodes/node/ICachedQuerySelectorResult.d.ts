import Element from '../element/Element.cjs';
import ICachedResult from './ICachedResult.cjs';
export default interface ICachedQuerySelectorResult extends ICachedResult {
    result: WeakRef<Element | null> | null;
}
//# sourceMappingURL=ICachedQuerySelectorResult.d.ts.map