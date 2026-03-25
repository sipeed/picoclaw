import Element from '../nodes/element/Element.cjs';
import SelectorCombinatorEnum from './SelectorCombinatorEnum.cjs';
import ISelectorAttribute from './ISelectorAttribute.cjs';
import ISelectorMatch from './ISelectorMatch.cjs';
import ISelectorPseudo from './ISelectorPseudo.cjs';
/**
 * Selector item.
 */
export default class SelectorItem {
    tagName: string | null;
    id: string | null;
    classNames: string[] | null;
    attributes: ISelectorAttribute[] | null;
    pseudos: ISelectorPseudo[] | null;
    isPseudoElement: boolean;
    combinator: SelectorCombinatorEnum;
    ignoreErrors: boolean;
    /**
     * Constructor.
     *
     * @param [options] Options.
     * @param [options.combinator] Combinator.
     * @param [options.tagName] Tag name.
     * @param [options.id] ID.
     * @param [options.classNames] Class names.
     * @param [options.attributes] Attributes.
     * @param [options.pseudos] Pseudos.
     * @param [options.isPseudoElement] Is pseudo element.
     * @param [options.ignoreErrors] Ignore errors.
     */
    constructor(options?: {
        tagName?: string;
        id?: string;
        classNames?: string[];
        attributes?: ISelectorAttribute[];
        pseudos?: ISelectorPseudo[];
        isPseudoElement?: boolean;
        combinator?: SelectorCombinatorEnum;
        ignoreErrors?: boolean;
    });
    /**
     * Matches a selector against an element.
     *
     * @param element HTML element.
     * @returns Result.
     */
    match(element: Element): ISelectorMatch | null;
    /**
     * Matches a pseudo selector.
     *
     * @param element Element.
     * @returns Result.
     */
    private matchPseudo;
    /**
     * Matches a pseudo selector.
     *
     * @param element Element.
     * @param parentChildren Parent children.
     * @param pseudo Pseudo.
     */
    private matchPseudoItem;
    /**
     * Matches attribute.
     *
     * @param element Element.
     * @returns Result.
     */
    private matchAttributes;
    /**
     * Matches class.
     *
     * @param element Element.
     * @returns Result.
     */
    private matchClass;
    /**
     * Matches a selector item against children of an element.
     *
     * @param selectorItem Selector item.
     * @param element Element.
     * @returns Result.
     */
    private matchChildOfElement;
    /**
     * Returns the selector string.
     *
     * @returns Selector string.
     */
    private getSelectorString;
}
//# sourceMappingURL=SelectorItem.d.ts.map