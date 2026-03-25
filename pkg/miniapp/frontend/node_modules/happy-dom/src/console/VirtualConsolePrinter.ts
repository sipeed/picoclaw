import IVirtualConsoleLogEntry from './IVirtualConsoleLogEntry.js';
import VirtualConsoleLogLevelEnum from './enums/VirtualConsoleLogLevelEnum.js';
import Event from '../event/Event.js';
import VirtualConsoleLogEntryStringifier from './utilities/VirtualConsoleLogEntryStringifier.js';
import IVirtualConsolePrinter from './IVirtualConsolePrinter.js';

/**
 * Virtual console printer.
 */
export default class VirtualConsolePrinter implements IVirtualConsolePrinter {
	#logEntries: IVirtualConsoleLogEntry[] = [];
	#listeners: {
		print: Array<(event: Event) => void>;
		clear: Array<(event: Event) => void>;
	} = { print: [], clear: [] };

	/**
	 * Writes to the output.
	 *
	 * @param logEntry Log entry.
	 */
	public print(logEntry: IVirtualConsoleLogEntry): void {
		this.#logEntries.push(logEntry);
		this.dispatchEvent(new Event('print'));
	}

	/**
	 * Clears the output.
	 */
	public clear(): void {
		this.#logEntries = [];
		this.dispatchEvent(new Event('clear'));
	}

	/**
	 * Adds an event listener.
	 *
	 * @param eventType Event type ("print" or "clear").
	 * @param listener Listener.
	 */
	public addEventListener(eventType: 'print' | 'clear', listener: (event: Event) => void): void {
		if (!this.#listeners[eventType]) {
			throw new Error(`Event type "${eventType}" is not supported.`);
		}
		this.#listeners[eventType].push(listener);
	}

	/**
	 * Removes an event listener.
	 *
	 * @param eventType Event type ("print" or "clear").
	 * @param listener Listener.
	 */
	public removeEventListener(eventType: 'print' | 'clear', listener: (event: Event) => void): void {
		if (!this.#listeners[eventType]) {
			throw new Error(`Event type "${eventType}" is not supported.`);
		}
		const index = this.#listeners[eventType].indexOf(listener);
		if (index !== -1) {
			this.#listeners[eventType].splice(index, 1);
		}
	}

	/**
	 * Dispatches an event.
	 *
	 * @param event Event.
	 */
	public dispatchEvent(event: Event): void {
		if (!this.#listeners[event.type]) {
			throw new Error(`Event type "${event.type}" is not supported.`);
		}
		for (const listener of this.#listeners[event.type]) {
			listener(event);
		}
	}

	/**
	 * Reads the buffer.
	 *
	 * @returns Console log entries.
	 */
	public read(): IVirtualConsoleLogEntry[] {
		const logEntries = this.#logEntries;
		this.#logEntries = [];
		return logEntries;
	}

	/**
	 * Returns the buffer as a string.
	 *
	 * @param [logLevel] Log level.
	 * @returns Buffer as a string of concatenated log entries.
	 */
	public readAsString(
		logLevel: VirtualConsoleLogLevelEnum = VirtualConsoleLogLevelEnum.log
	): string {
		const logEntries = this.read();
		let output = '';
		for (const logEntry of logEntries) {
			if (logEntry.level >= logLevel) {
				output += VirtualConsoleLogEntryStringifier.toString(logEntry);
			}
		}
		return output;
	}
}
