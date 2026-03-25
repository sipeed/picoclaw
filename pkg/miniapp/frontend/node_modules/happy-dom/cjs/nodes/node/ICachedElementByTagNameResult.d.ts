import Element from '../element/Element.cjs';
import ICachedResult from './ICachedResult.cjs';
export default interface ICachedElementByTagNameResult extends ICachedResult {
    result: WeakRef<Element> | null;
}
//# sourceMappingURL=ICachedElementByTagNameResult.d.ts.map