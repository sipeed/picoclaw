import Element from '../element/Element.cjs';
import ICachedResult from './ICachedResult.cjs';
export default interface ICachedElementsByTagNameResult extends ICachedResult {
    result: WeakRef<Element[]> | null;
}
//# sourceMappingURL=ICachedElementsByTagNameResult.d.ts.map