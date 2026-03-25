import HTMLInputElement from './HTMLInputElement.js';

const NEW_LINES_REGEXP = /[\n\r]/gm;
const parseInts = (a: string[]): number[] => a.map((v) => parseInt(v, 10));

/**
 * HTML input element value sanitizer.
 */
export default class HTMLInputElementValueSanitizer {
	/**
	 * Sanitizes a value.
	 *
	 * @param input Input.
	 * @param value Value.
	 */
	public static sanitize(input: HTMLInputElement, value: string): string {
		switch (input.type) {
			case 'password':
			case 'search':
			case 'tel':
			case 'text':
				return value.replace(NEW_LINES_REGEXP, '');
			case 'color':
				// https://html.spec.whatwg.org/multipage/forms.html#color-state-(type=color):value-sanitization-algorithm
				return /^#[a-fA-F\d]{6}$/.test(value) ? value.toLowerCase() : '#000000';
			case 'email':
				// https://html.spec.whatwg.org/multipage/forms.html#e-mail-state-(type=email):value-sanitization-algorithm
				// https://html.spec.whatwg.org/multipage/forms.html#e-mail-state-(type=email):value-sanitization-algorithm-2
				if (input.multiple) {
					return value
						.split(',')
						.map((token) => token.trim())
						.join(',');
				}
				return value.trim().replace(NEW_LINES_REGEXP, '');
			case 'number':
				// https://html.spec.whatwg.org/multipage/input.html#number-state-(type=number):value-sanitization-algorithm
				return !isNaN(Number.parseFloat(value)) ? value : '';
			case 'range': {
				// https://html.spec.whatwg.org/multipage/input.html#range-state-(type=range):value-sanitization-algorithm
				const number = Number.parseFloat(value);
				const min = parseFloat(input.min) || 0;
				const max = parseFloat(input.max) || 100;

				if (isNaN(number)) {
					return max < min ? String(min) : String((min + max) / 2);
				} else if (number < min) {
					return String(min);
				} else if (number > max) {
					return String(max);
				}

				return value;
			}
			case 'url':
				// https://html.spec.whatwg.org/multipage/forms.html#url-state-(type=url):value-sanitization-algorithm
				return value.trim().replace(NEW_LINES_REGEXP, '');
			case 'date':
				// https://html.spec.whatwg.org/multipage/input.html#date-state-(type=date):value-sanitization-algorithm
				value = this.sanitizeDate(value);
				return value && this.checkBoundaries(value, input.min, input.max) ? value : '';
			case 'datetime-local': {
				// https://html.spec.whatwg.org/multipage/input.html#local-date-and-time-state-(type=datetime-local):value-sanitization-algorithm
				const match = value.match(
					/^(\d\d\d\d)-(\d\d)-(\d\d)[T ](\d\d):(\d\d)(?::(\d\d)(?:\.(\d{1,3}))?)?$/
				);
				if (!match) {
					return '';
				}
				const dateString = this.sanitizeDate(value.slice(0, 10));
				let timeString = this.sanitizeTime(value.slice(11));
				if (!(dateString && timeString)) {
					return '';
				}
				// Has seconds so needs to remove trailing zeros
				if (match[6] !== undefined) {
					if (timeString.indexOf('.') !== -1) {
						// Remove unecessary zeros milliseconds
						timeString = timeString.replace(/(?:\.0*|(\.\d+?)0+)$/, '$1');
					}
					timeString = timeString.replace(/(\d\d:\d\d)(:00)$/, '$1');
				}
				return dateString + 'T' + timeString;
			}
			case 'month':
				// https://html.spec.whatwg.org/multipage/input.html#month-state-(type=month):value-sanitization-algorithm
				if (!(value.match(/^(\d\d\d\d)-(\d\d)$/) && this.parseMonthComponent(value))) {
					return '';
				}
				return this.checkBoundaries(value, input.min, input.max) ? value : '';
			case 'time': {
				// https://html.spec.whatwg.org/multipage/input.html#time-state-(type=time):value-sanitization-algorithm
				value = this.sanitizeTime(value);
				return value && this.checkBoundaries(value, input.min, input.max) ? value : '';
			}
			case 'week': {
				// https://html.spec.whatwg.org/multipage/input.html#week-state-(type=week):value-sanitization-algorithm
				const match = value.match(/^(\d\d\d\d)-W(\d\d)$/);
				if (!match) {
					return '';
				}
				const [intY, intW] = parseInts(match.slice(1, 3));
				if (intY <= 0 || intW < 1 || intW > 53) {
					return '';
				}
				// Check date is valid
				const lastWeek = this.lastIsoWeekOfYear(intY);
				if (intW < 1 || intW > 52 + lastWeek) {
					return '';
				}
				if (!this.checkBoundaries(value, input.min, input.max)) {
					return '';
				}
				return value;
			}
		}

		return value;
	}
	/**
	 * Checks if a value is within the boundaries of min and max.
	 *
	 * @param value
	 * @param min
	 * @param max
	 */
	private static checkBoundaries<T>(value: T, min: T, max: T): boolean {
		if (min && min > value) {
			return false;
		} else if (max && max < value) {
			return false;
		}
		return true;
	}
	/**
	 * Parses the month component of a date string.
	 *
	 * @param value
	 */
	private static parseMonthComponent(value: string): string {
		const [Y, M] = value.split('-');
		const [intY, intM] = parseInts([Y, M]);
		if (isNaN(intY) || isNaN(intM) || intY <= 0 || intM < 1 || intM > 12) {
			return '';
		}
		return value;
	}
	/**
	 * Returns the last ISO week of a year.
	 *
	 * @param year
	 */
	private static lastIsoWeekOfYear = (year: string | number): number => {
		const date = new Date(+year, 11, 31);
		const day = (date.getDay() + 6) % 7;
		date.setDate(date.getDate() - day + 3);
		const firstThursday = date.getTime();
		date.setMonth(0, 1);
		if (date.getDay() !== 4) {
			date.setMonth(0, 1 + ((4 - date.getDay() + 7) % 7));
		}
		return 1 + Math.ceil((firstThursday - date.getTime()) / 604800000);
	};

	/**
	 * Sanitizes a date string.
	 *
	 * @param value
	 */
	private static sanitizeDate(value: string): string {
		const match = value.match(/^(\d{4})-(\d{2})-(\d{2})$/);
		if (!match) {
			return '';
		}
		const month = this.parseMonthComponent(value.slice(0, 7));
		if (!month) {
			return '';
		}
		const [intY, intM, intD] = parseInts(match.slice(1, 4));
		if (intD < 1 || intD > 31) {
			return '';
		}
		// Check date is valid
		const lastDayOfMonth = new Date(intY, intM, 0).getDate();
		if (intD > lastDayOfMonth) {
			return '';
		}
		return value;
	}

	/**
	 * Sanitizes a time string.
	 *
	 * @param value
	 */
	private static sanitizeTime(value: string): string {
		const match = value.match(/^(\d{2}):(\d{2})(?::(\d{2}(?:\.(\d{1,3}))?))?$/);
		if (!match) {
			return '';
		}
		const [intH, intM] = parseInts(match.slice(1, 3));
		const ms = parseFloat(match[3] || '0') * 1000;
		if (intH > 23 || intM > 59 || ms > 59999) {
			return '';
		}
		if (ms === 0) {
			return `${match[1]}:${match[2]}`;
		} else {
			return `${match[1]}:${match[2]}${ms >= 10000 ? `:${ms / 1000}` : `:0${ms / 1000}`}`;
		}
	}
}
