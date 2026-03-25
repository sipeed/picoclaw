import IVirtualConsoleLogEntry from '../IVirtualConsoleLogEntry.js';
import VirtualConsoleLogTypeEnum from '../enums/VirtualConsoleLogTypeEnum.js';

/**
 * Virtual console utility.
 */
export default class VirtualConsoleLogEntryStringifier {
	/**
	 * Stringifies a log entry.
	 *
	 * @param logEntry Log entry.
	 * @returns Stringified message.
	 */
	public static toString(logEntry: IVirtualConsoleLogEntry): string {
		if (this.isLogEntryCollapsed(logEntry)) {
			return '';
		}

		const tabbing = this.getLogEntryGroupTabbing(logEntry);
		let output = tabbing;
		for (const part of logEntry.message) {
			output += output !== '' && output !== tabbing ? ' ' : '';
			if (
				typeof part === 'object' &&
				(part === null || part.constructor.name === 'Object' || Array.isArray(part))
			) {
				try {
					output += JSON.stringify(part);
				} catch (error) {
					output += new Error('Failed to JSON stringify object in log entry.').stack.replace(
						/\n    at/gm,
						'\n    ' + tabbing + 'at'
					);
				}
			} else if (typeof part === 'object' && part['message'] && part['stack']) {
				output += part['stack'].replace(/\n    at/gm, '\n    ' + tabbing + 'at');
			} else {
				output += this.getLogEntryIcon(logEntry) + String(part);
			}
		}
		return output + '\n';
	}

	/**
	 * Gets the log entry icon.
	 *
	 * @param logEntry Log entry.
	 * @returns Icon.
	 */
	private static getLogEntryIcon(logEntry: IVirtualConsoleLogEntry): string {
		switch (logEntry.type) {
			case VirtualConsoleLogTypeEnum.group:
				return '▼ ';
			case VirtualConsoleLogTypeEnum.groupCollapsed:
				return '▶ ';
		}
		return '';
	}

	/**
	 * Gets the log entry group tabbing.
	 *
	 * @param logEntry Log entry.
	 * @returns Tabbing.
	 */
	private static getLogEntryGroupTabbing(logEntry: IVirtualConsoleLogEntry): string {
		let tabs = '';
		let group =
			logEntry.type === VirtualConsoleLogTypeEnum.group ||
			logEntry.type === VirtualConsoleLogTypeEnum.groupCollapsed
				? logEntry.group?.parent
				: logEntry.group;
		while (group) {
			tabs += '  ';
			group = group.parent;
		}
		return tabs;
	}

	/**
	 * Checks if the log entry content is collapsed.
	 *
	 * @param logEntry Log entry.
	 * @returns True if collapsed.
	 */
	private static isLogEntryCollapsed(logEntry: IVirtualConsoleLogEntry): boolean {
		let group =
			logEntry.type === VirtualConsoleLogTypeEnum.group ||
			logEntry.type === VirtualConsoleLogTypeEnum.groupCollapsed
				? logEntry.group?.parent
				: logEntry.group;
		while (group) {
			if (group.collapsed) {
				return true;
			}
			group = group.parent;
		}
		return false;
	}
}
