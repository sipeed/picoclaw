import SelectorItem from './SelectorItem.js';
export default interface ISelectorPseudo {
    name: string;
    arguments: string | null;
    selectorItems: SelectorItem[] | null;
    nthFunction: ((n: number) => boolean) | null;
}
//# sourceMappingURL=ISelectorPseudo.d.ts.map