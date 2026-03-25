import ISelectorMatch from '../../query-selector/ISelectorMatch.cjs';
import ICachedResult from './ICachedResult.cjs';
export default interface ICachedMatchesResult extends ICachedResult {
    result: {
        match: ISelectorMatch | null;
    } | null;
}
//# sourceMappingURL=ICachedMatchesResult.d.ts.map