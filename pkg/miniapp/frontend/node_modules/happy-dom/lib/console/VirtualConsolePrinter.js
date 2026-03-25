import VirtualConsoleLogLevelEnum from './enums/VirtualConsoleLogLevelEnum.js';
import Event from '../event/Event.js';
import VirtualConsoleLogEntryStringifier from './utilities/VirtualConsoleLogEntryStringifier.js';
/**
 * Virtual console printer.
 */
export default class VirtualConsolePrinter {
    #logEntries = [];
    #listeners = { print: [], clear: [] };
    /**
     * Writes to the output.
     *
     * @param logEntry Log entry.
     */
    print(logEntry) {
        this.#logEntries.push(logEntry);
        this.dispatchEvent(new Event('print'));
    }
    /**
     * Clears the output.
     */
    clear() {
        this.#logEntries = [];
        this.dispatchEvent(new Event('clear'));
    }
    /**
     * Adds an event listener.
     *
     * @param eventType Event type ("print" or "clear").
     * @param listener Listener.
     */
    addEventListener(eventType, listener) {
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
    removeEventListener(eventType, listener) {
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
    dispatchEvent(event) {
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
    read() {
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
    readAsString(logLevel = VirtualConsoleLogLevelEnum.log) {
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
//# sourceMappingURL=VirtualConsolePrinter.js.map