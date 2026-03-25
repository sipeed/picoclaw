import CSSStyleDeclarationPropertyManager from '../../css/declaration/property-manager/CSSStyleDeclarationPropertyManager.cjs';
import ICachedResult from './ICachedResult.cjs';
export default interface ICachedStyleResult extends ICachedResult {
    result: WeakRef<CSSStyleDeclarationPropertyManager | null> | null;
}
//# sourceMappingURL=ICachedStyleResult.d.ts.map