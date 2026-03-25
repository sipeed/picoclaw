import SelectorItem from './SelectorItem.js';
import SelectorCombinatorEnum from './SelectorCombinatorEnum.js';
import DOMException from '../exception/DOMException.js';
import ISelectorPseudo from './ISelectorPseudo.js';

/**
 * Selector RegExp.
 *
 * Group 1: All (e.g. "*")
 * Group 2: Tag name (e.g. "div")
 * Group 3: ID (e.g. "#id")
 * Group 4: Class (e.g. ".class")
 * Group 5: Attribute name when no value (e.g. "attr1")
 * Group 6: Attribute name when there is a value using apostrophe (e.g. "attr1")
 * Group 7: Attribute operator when using apostrophe (e.g. "~")
 * Group 8: Attribute value when using apostrophe (e.g. "value1")
 * Group 9: Attribute modifier when using apostrophe (e.g. "i" or "s")
 * Group 10: Attribute name when threre is a value not using apostrophe (e.g. "attr1")
 * Group 11: Attribute operator when not using apostrophe (e.g. "~")
 * Group 12: Attribute value when notusing apostrophe (e.g. "value1")
 * Group 13: Pseudo name when arguments (e.g. "nth-child")
 * Group 14: Arguments of pseudo (e.g. "2n + 1")
 * Group 15: Pseudo name when no arguments (e.g. "empty")
 * Group 16: Pseudo element (e.g. "::after", "::-webkit-inner-spin-button").
 * Group 17: Combinator.
 */
const SELECTOR_REGEXP =
	/(\*)|([a-zA-Z0-9-]+)|#((?:[a-zA-Z0-9-_]|\\.)+)|\.((?:[a-zA-Z0-9-_]|\\.)+)|\[([a-zA-Z0-9-_\\:]+)\]|\[([a-zA-Z0-9-_\\:]+)\s*([~|^$*]{0,1})\s*=\s*["']{1}([^"']*)["']{1}\s*(s|i){0,1}\]|\[([a-zA-Z0-9-_]+)\s*([~|^$*]{0,1})\s*=\s*([^\]]*)\]|:([a-zA-Z-]+)\s*\(([^)]+)\){0,1}|:([a-zA-Z-]+)|::([a-zA-Z-]+)|([\s,+>~]*)/gm;

/**
 * Escaped Character RegExp.
 */
const ESCAPED_CHARACTER_REGEXP = /\\/g;

/**
 * Nth Function.
 */
const NTH_FUNCTION = {
	odd: (n: number) => (n + 1) % 2 === 0,
	even: (n: number) => (n + 1) % 2 !== 0,
	alwaysFalse: () => false
};

/**
 * Space RegExp.
 */
const SPACE_REGEXP = / /g;

/**
 * Simple Selector RegExp.
 *
 * Group 1: Tag name (e.g. "div")
 * Group 2: Class (e.g. ".classA.classB")
 * Group 3: ID (e.g. "#id")
 */
const SIMPLE_SELECTOR_REGEXP = /(^[a-zA-Z0-9-]+$)|(^\.[a-zA-Z0-9-_.]+$)|(^#[a-zA-Z0-9-_]+$)/;

/**
 * Utility for parsing a selection string.
 */
export default class SelectorParser {
	/**
	 * Parses a selector string and returns an instance of SelectorItem.
	 *
	 * @param selector Selector.
	 * @param [options] Options.
	 * @param [options.ignoreErrors] Ignores errors.
	 * @returns Selector item.
	 */
	public static getSelectorItem(
		selector: string,
		options?: { ignoreErrors?: boolean }
	): SelectorItem {
		return this.getSelectorGroups(selector, options)[0][0];
	}

	/**
	 * Parses a selector string and returns groups with SelectorItem instances.
	 *
	 * @param selector Selector.
	 * @param [options] Options.
	 * @param [options.ignoreErrors] Ignores errors.
	 * @returns Selector groups.
	 */
	public static getSelectorGroups(
		selector: string,
		options?: { ignoreErrors?: boolean }
	): Array<Array<SelectorItem>> {
		selector = selector.trim();
		const ignoreErrors = options?.ignoreErrors;

		if (selector === '*') {
			return [[new SelectorItem({ tagName: '*', ignoreErrors })]];
		}

		const simpleMatch = selector.match(SIMPLE_SELECTOR_REGEXP);

		if (simpleMatch) {
			if (simpleMatch[1]) {
				return [[new SelectorItem({ tagName: selector.toUpperCase(), ignoreErrors })]];
			} else if (simpleMatch[2]) {
				return [
					[new SelectorItem({ classNames: selector.replace('.', '').split('.'), ignoreErrors })]
				];
			} else if (simpleMatch[3]) {
				return [[new SelectorItem({ id: selector.replace('#', ''), ignoreErrors })]];
			}
		}

		const regexp = new RegExp(SELECTOR_REGEXP);
		let currentSelectorItem: SelectorItem = new SelectorItem({
			combinator: SelectorCombinatorEnum.descendant,
			ignoreErrors
		});
		let currentGroup: SelectorItem[] = [currentSelectorItem];
		const groups: Array<Array<SelectorItem>> = [currentGroup];
		let isValid = false;
		let match;

		while ((match = regexp.exec(selector))) {
			if (match[0]) {
				isValid = true;

				if (match[1]) {
					currentSelectorItem.tagName = '*';
				} else if (match[2]) {
					currentSelectorItem.tagName = match[2].toUpperCase();
				} else if (match[3]) {
					currentSelectorItem.id = match[3].replace(ESCAPED_CHARACTER_REGEXP, '');
				} else if (match[4]) {
					currentSelectorItem.classNames = currentSelectorItem.classNames || [];
					currentSelectorItem.classNames.push(match[4].replace(ESCAPED_CHARACTER_REGEXP, ''));
				} else if (match[5]) {
					currentSelectorItem.attributes = currentSelectorItem.attributes || [];
					currentSelectorItem.attributes.push({
						name: match[5].toLowerCase(),
						operator: null,
						value: null,
						modifier: null,
						regExp: null
					});
				} else if (match[6] && match[8] !== undefined) {
					currentSelectorItem.attributes = currentSelectorItem.attributes || [];
					currentSelectorItem.attributes.push({
						name: match[6].toLowerCase(),
						operator: match[7] || null,
						value: match[8].replace(ESCAPED_CHARACTER_REGEXP, ''),
						modifier: match[9] || null,
						regExp: this.getAttributeRegExp({
							operator: match[7],
							value: match[8],
							modifier: match[9]
						})
					});
				} else if (match[10] && match[12] !== undefined) {
					currentSelectorItem.attributes = currentSelectorItem.attributes || [];
					currentSelectorItem.attributes.push({
						name: match[10].toLowerCase(),
						operator: match[11] || null,
						value: match[12].replace(ESCAPED_CHARACTER_REGEXP, ''),
						modifier: null,
						regExp: this.getAttributeRegExp({ operator: match[11], value: match[12] })
					});
				} else if (match[13] && match[14]) {
					currentSelectorItem.pseudos = currentSelectorItem.pseudos || [];
					currentSelectorItem.pseudos.push(this.getPseudo(match[13], match[14], options));
				} else if (match[15]) {
					currentSelectorItem.pseudos = currentSelectorItem.pseudos || [];
					currentSelectorItem.pseudos.push(this.getPseudo(match[15], null, options));
				} else if (match[16]) {
					currentSelectorItem.isPseudoElement = true;
				} else if (match[17]) {
					switch (match[17].trim()) {
						case ',':
							currentSelectorItem = new SelectorItem({
								combinator: SelectorCombinatorEnum.descendant,
								ignoreErrors
							});
							currentGroup = [currentSelectorItem];
							groups.push(currentGroup);
							break;
						case '>':
							currentSelectorItem = new SelectorItem({
								combinator: SelectorCombinatorEnum.child,
								ignoreErrors
							});
							currentGroup.push(currentSelectorItem);
							break;
						case '+':
							currentSelectorItem = new SelectorItem({
								combinator: SelectorCombinatorEnum.adjacentSibling,
								ignoreErrors
							});
							currentGroup.push(currentSelectorItem);
							break;
						case '~':
							currentSelectorItem = new SelectorItem({
								combinator: SelectorCombinatorEnum.subsequentSibling,
								ignoreErrors
							});
							currentGroup.push(currentSelectorItem);
							break;
						case '':
							currentSelectorItem = new SelectorItem({
								combinator: SelectorCombinatorEnum.descendant,
								ignoreErrors
							});
							currentGroup.push(currentSelectorItem);
							break;
					}
				}
			} else {
				break;
			}
		}

		if (!isValid) {
			if (options?.ignoreErrors) {
				return [];
			}
			throw new DOMException(`Invalid selector: "${selector}"`);
		}

		return groups;
	}

	/**
	 * Returns attribute RegExp.
	 *
	 * @param attribute Attribute.
	 * @param attribute.value Attribute value.
	 * @param attribute.operator Attribute operator.
	 * @param attribute.modifier Attribute modifier.
	 * @returns Attribute RegExp.
	 */
	private static getAttributeRegExp(attribute: {
		value?: string;
		operator?: string;
		modifier?: string;
	}): RegExp | null {
		const modifier = attribute.modifier === 'i' ? 'i' : '';

		if (!attribute.operator || !attribute.value) {
			return null;
		}

		switch (attribute.operator) {
			// [attribute~="value"] - Contains a specified word.
			case '~':
				return new RegExp(
					`[- ]${attribute.value}|${attribute.value}[- ]|^${attribute.value}$`,
					modifier
				);
			// [attribute|="value"] - Starts with the specified word.
			case '|':
				return new RegExp(`^${attribute.value}[- ]|^${attribute.value}$`, modifier);
			// [attribute^="value"] - Begins with a specified value.
			case '^':
				return new RegExp(`^${attribute.value}`, modifier);
			// [attribute$="value"] - Ends with a specified value.
			case '$':
				return new RegExp(`${attribute.value}$`, modifier);
			// [attribute*="value"] - Contains a specified value.
			case '*':
				return new RegExp(`${attribute.value}`, modifier);
			default:
				return null;
		}
	}

	/**
	 * Returns pseudo.
	 *
	 * @param name Pseudo name.
	 * @param args Pseudo arguments.
	 * @param [options] Options.
	 * @param [options.ignoreErrors] Ignores errors.
	 * @returns Pseudo.
	 */
	private static getPseudo(
		name: string,
		args?: string,
		options?: { ignoreErrors?: boolean }
	): ISelectorPseudo {
		const lowerName = name.toLowerCase();

		if (args) {
			args = args.trim();
		}

		if (!args) {
			return { name: lowerName, arguments: null, selectorItems: null, nthFunction: null };
		}

		switch (lowerName) {
			case 'nth-last-child':
			case 'nth-child':
				const nthOfIndex = args.indexOf(' of ');
				const nthFunction = nthOfIndex !== -1 ? args.substring(0, nthOfIndex) : args;
				const selectorItem =
					nthOfIndex !== -1
						? this.getSelectorItem(args.substring(nthOfIndex + 4).trim(), options)
						: null;
				return {
					name: lowerName,
					arguments: args,
					selectorItems: [selectorItem],
					nthFunction: this.getPseudoNthFunction(nthFunction)
				};
			case 'nth-of-type':
			case 'nth-last-of-type':
				return {
					name: lowerName,
					arguments: args,
					selectorItems: null,
					nthFunction: this.getPseudoNthFunction(args)
				};
			case 'not':
				return {
					name: lowerName,
					arguments: args,
					selectorItems: [this.getSelectorItem(args, options)],
					nthFunction: null
				};
			case 'is':
			case 'where':
				const selectorItems = [];
				for (const group of this.getSelectorGroups(args, options)) {
					selectorItems.push(group[0]);
				}
				return {
					name: lowerName,
					arguments: args,
					selectorItems,
					nthFunction: null
				};
			case 'has':
				const hasSelectorItems = [];

				// The ":has()" pseudo selector doesn't allow for it to be nested inside another ":has()" pseudo selector, as it can lead to cyclic querying.
				if (!args.includes(':has(')) {
					let newArgs = args;
					if (args[0] === '+') {
						newArgs = args.replace('+', '');
					} else if (args[0] === '>') {
						newArgs = args.replace('>', '');
					}
					for (const group of this.getSelectorGroups(newArgs, options)) {
						hasSelectorItems.push(group[0]);
					}
				}

				return {
					name: lowerName,
					arguments: args,
					selectorItems: hasSelectorItems,
					nthFunction: null
				};
			default:
				return { name: lowerName, arguments: args, selectorItems: null, nthFunction: null };
		}
	}

	/**
	 * Returns pseudo nth function.
	 *
	 * Based on:
	 * https://github.com/dperini/nwsapi/blob/master/src/nwsapi.js
	 *
	 * @param args Pseudo arguments.
	 * @returns Pseudo nth function.
	 */
	private static getPseudoNthFunction(args?: string): ((n: number) => boolean) | null {
		if (args === 'odd') {
			return NTH_FUNCTION.odd;
		} else if (args === 'even') {
			return NTH_FUNCTION.even;
		}

		const parts = args.replace(SPACE_REGEXP, '').split('n');
		let partA = parseInt(parts[0], 10) || 0;

		if (parts[0] == '-') {
			partA = -1;
		}

		if (parts.length === 1) {
			return (n) => n == partA;
		}

		let partB = parseInt(parts[1], 10) || 0;

		if (parts[0] == '+') {
			partB = 1;
		}

		if (partA >= 1 || partA <= -1) {
			if (partA >= 1) {
				if (Math.abs(partA) === 1) {
					return (n: number): boolean => n > partB - 1;
				}
				return (n: number): boolean => n > partB - 1 && (n + -1 * partB) % partA === 0;
			}
			if (Math.abs(partA) === 1) {
				return (n: number): boolean => n < partB + 1;
			}
			return (n) => n < partB + 1 && (n + -1 * partB) % partA === 0;
		}

		if (parts[0]) {
			return (n) => n === partB;
		}

		return (n) => n > partB - 1;
	}
}
