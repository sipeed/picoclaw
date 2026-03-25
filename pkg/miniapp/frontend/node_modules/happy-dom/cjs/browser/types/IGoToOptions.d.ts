import IRequestReferrerPolicy from '../../fetch/types/IRequestReferrerPolicy.cjs';
import IReloadOptions from './IReloadOptions.cjs';
/**
 * Go to options.
 */
export default interface IGoToOptions extends IReloadOptions {
    /**
     * Referrer.
     */
    referrer?: string;
    /**
     * Referrer policy.
     */
    referrerPolicy?: IRequestReferrerPolicy;
}
//# sourceMappingURL=IGoToOptions.d.ts.map