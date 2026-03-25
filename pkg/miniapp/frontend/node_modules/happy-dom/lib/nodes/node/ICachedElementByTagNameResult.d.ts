import Element from '../element/Element.js';
import ICachedResult from './ICachedResult.js';
export default interface ICachedElementByTagNameResult extends ICachedResult {
    result: WeakRef<Element> | null;
}
//# sourceMappingURL=ICachedElementByTagNameResult.d.ts.map