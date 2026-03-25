import Element from '../element/Element.js';
import ICachedResult from './ICachedResult.js';
export default interface ICachedElementByIdResult extends ICachedResult {
    result: WeakRef<Element> | null;
}
//# sourceMappingURL=ICachedElementByIdResult.d.ts.map