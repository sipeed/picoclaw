import EventTarget from '../../event/EventTarget.js';
import Event from '../../event/Event.js';
import TextTrackCue from './TextTrackCue.js';
import TextTrackCueList from './TextTrackCueList.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import TextTrackKindEnum from './TextTrackKindEnum.js';

/**
 * TextTrack.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/TextTrack
 */
export default class TextTrack extends EventTarget {
	// Internal properties
	public [PropertySymbol.kind]: TextTrackKindEnum = TextTrackKindEnum.subtitles;
	public [PropertySymbol.label]: string = '';
	public [PropertySymbol.language]: string = '';
	public [PropertySymbol.id]: string = '';
	public [PropertySymbol.mode]: 'disabled' | 'showing' = 'disabled';
	public [PropertySymbol.cues]: TextTrackCueList = new TextTrackCueList(
		PropertySymbol.illegalConstructor
	);
	public [PropertySymbol.activeCues]: TextTrackCueList = new TextTrackCueList(
		PropertySymbol.illegalConstructor
	);

	// Events
	public oncuechange: (event: Event) => void = null;

	/**
	 * Constructor.
	 *
	 * @param illegalConstructorSymbol Illegal constructor symbol.
	 */
	constructor(illegalConstructorSymbol: symbol) {
		super();

		if (illegalConstructorSymbol !== PropertySymbol.illegalConstructor) {
			throw new TypeError('Illegal constructor');
		}

		if (!this[PropertySymbol.window]) {
			throw new TypeError(
				`Failed to construct '${this.constructor.name}': '${this.constructor.name}' was constructed outside a Window context.`
			);
		}
	}

	/**
	 * Returns the kind of the text track.
	 *
	 * @returns Kind.
	 */
	public get kind(): TextTrackKindEnum {
		return this[PropertySymbol.kind];
	}

	/**
	 * Returns the label of the text track.
	 *
	 * @returns Label.
	 */
	public get label(): string {
		return this[PropertySymbol.label];
	}

	/**
	 * Returns the language of the text track.
	 *
	 * @returns Language.
	 */
	public get language(): string {
		return this[PropertySymbol.language];
	}

	/**
	 * Returns the id of the text track.
	 *
	 * @returns Id.
	 */
	public get id(): string {
		return this[PropertySymbol.id];
	}

	/**
	 * Returns the mode of the text track.
	 *
	 * @returns Mode.
	 */
	public get mode(): 'disabled' | 'showing' {
		return this[PropertySymbol.mode];
	}

	/**
	 * Sets the mode of the text track.
	 *
	 * @param mode Mode.
	 */
	public set mode(mode: 'disabled' | 'showing') {
		if (mode !== 'disabled' && mode !== 'showing') {
			// TODO: Browser outputs a warning here.
			return;
		}
		this[PropertySymbol.mode] = mode;
	}

	/**
	 * Returns the list of cues in the track list.
	 *
	 * @returns List of cues.
	 */
	public get cues(): TextTrackCueList | null {
		if (this[PropertySymbol.mode] === 'disabled') {
			return null;
		}
		return this[PropertySymbol.cues];
	}

	/**
	 * Returns the list of active cues in the track list.
	 *
	 * @returns List of active cues.
	 */
	public get activeCues(): TextTrackCueList | null {
		if (this[PropertySymbol.mode] === 'disabled') {
			return null;
		}
		return this[PropertySymbol.activeCues];
	}

	/**
	 * Adds a cue to the track list.
	 *
	 * @param cue Text track cue.
	 */
	public addCue(cue: TextTrackCue): void {
		if (this[PropertySymbol.cues].includes(cue)) {
			return;
		}
		cue[PropertySymbol.track] = this;
		this[PropertySymbol.cues].push(cue);
	}

	/**
	 * Removes a cue from the track list.
	 *
	 * @param cue Text track cue.
	 */
	public removeCue(cue: TextTrackCue): void {
		const index = this[PropertySymbol.cues].indexOf(cue);
		if (index !== -1) {
			cue[PropertySymbol.track] = null;
			this[PropertySymbol.cues].splice(index, 1);
		}
	}
}
