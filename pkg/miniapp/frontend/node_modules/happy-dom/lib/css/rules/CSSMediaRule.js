import CSSRule from '../CSSRule.js';
import MediaList from '../MediaList.js';
/**
 * CSSRule interface.
 */
export default class CSSMediaRule extends CSSRule {
    type = CSSRule.MEDIA_RULE;
    cssRules = [];
    media = new MediaList();
    /**
     * Returns css text.
     *
     * @returns CSS text.
     */
    get cssText() {
        let cssText = '';
        for (const cssRule of this.cssRules) {
            cssText += cssRule.cssText;
        }
        return `@media ${this.conditionText} { ${cssText} }`;
    }
    /**
     * Returns conditional text.
     *
     * @returns Conditional text.
     */
    get conditionText() {
        return this.media.mediaText;
    }
}
//# sourceMappingURL=CSSMediaRule.js.map