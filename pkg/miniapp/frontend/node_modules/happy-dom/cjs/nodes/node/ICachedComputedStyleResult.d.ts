import CSSStyleDeclarationPropertyManager from '../../css/declaration/property-manager/CSSStyleDeclarationPropertyManager.cjs';
import ICachedResult from './ICachedResult.cjs';
export default interface ICachedComputedStyleResult extends ICachedResult {
    result: WeakRef<CSSStyleDeclarationPropertyManager | null> | null;
}
//# sourceMappingURL=ICachedComputedStyleResult.d.ts.map