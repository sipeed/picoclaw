import Element from '../element/Element.js';
import ICachedResult from './ICachedResult.js';
export default interface ICachedQuerySelectorResult extends ICachedResult {
    result: WeakRef<Element | null> | null;
}
//# sourceMappingURL=ICachedQuerySelectorResult.d.ts.map