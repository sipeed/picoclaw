import Element from '../element/Element.cjs';
import ICachedResult from './ICachedResult.cjs';
export default interface ICachedElementByIdResult extends ICachedResult {
    result: WeakRef<Element> | null;
}
//# sourceMappingURL=ICachedElementByIdResult.d.ts.map