import ISelectorMatch from '../../query-selector/ISelectorMatch.js';
import ICachedResult from './ICachedResult.js';
export default interface ICachedMatchesResult extends ICachedResult {
    result: {
        match: ISelectorMatch | null;
    } | null;
}
//# sourceMappingURL=ICachedMatchesResult.d.ts.map