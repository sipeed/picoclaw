import Event from '../../event/Event.js';
import EventTarget from '../../event/EventTarget.js';
import * as PropertySymbol from '../../PropertySymbol.js';
import Crypto from 'crypto';
import IMediaTrackCapabilities from './IMediaTrackCapabilities.js';
import IMediaTrackSettings from './IMediaTrackSettings.js';
const CAPABILITIES: IMediaTrackCapabilities = {
	aspectRatio: {
		max: 300,
		min: 0.006666666666666667
	},
	deviceId: '',
	facingMode: [],
	frameRate: {
		max: 60,
		min: 0
	},
	height: {
		max: 150,
		min: 1
	},
	resizeMode: ['none', 'crop-and-scale'],
	width: {
		max: 300,
		min: 1
	}
};
const SETTINGS: IMediaTrackSettings = {
	deviceId: '',
	frameRate: 60,
	resizeMode: 'none'
};

/**
 * Canvas Capture Media Stream Track.
 *
 * @see https://developer.mozilla.org/en-US/docs/Web/API/MediaStreamTrack
 */
export default class MediaStreamTrack extends EventTarget {
	public contentHint:
		| ''
		| 'speech'
		| 'speech-recognition'
		| 'music'
		| 'motion'
		| 'detail'
		| 'text' = '';
	public enabled = true;
	public readonly id: string = Crypto.randomUUID();
	public muted = false;
	public readyState: 'live' | 'ended' = 'live';
	public label: string = '';
	public [PropertySymbol.label]: string = '';
	public [PropertySymbol.kind]: 'audio' | 'video' = 'video';
	public [PropertySymbol.constraints]: object = {};
	public [PropertySymbol.capabilities]: IMediaTrackCapabilities = JSON.parse(
		JSON.stringify(CAPABILITIES)
	);
	public [PropertySymbol.settings]: IMediaTrackSettings = JSON.parse(JSON.stringify(SETTINGS));

	// Events
	public onended: ((event: Event) => void) | null = null;
	public onmute: ((event: Event) => void) | null = null;
	public onunmute: ((event: Event) => void) | null = null;

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
	 * Returns the kind of the track.
	 *
	 * @returns Kind.
	 */
	public get kind(): 'audio' | 'video' {
		return this[PropertySymbol.kind];
	}

	/**
	 * Applies constraints.
	 *
	 * @param _constraints Constraints.
	 * @param constraints
	 */
	public async applyConstraints(constraints: object): Promise<void> {
		this.#mergeObjects(this[PropertySymbol.constraints], constraints);
	}

	/**
	 * Returns constraints.
	 *
	 * @returns Constraints.
	 */
	public getConstraints(): object {
		return this[PropertySymbol.constraints];
	}

	/**
	 * Returns capabilities.
	 *
	 * @returns Capabilities.
	 */
	public getCapabilities(): IMediaTrackCapabilities {
		return this[PropertySymbol.capabilities];
	}

	/**
	 * Returns settings.
	 *
	 * @returns Settings.
	 */
	public getSettings(): IMediaTrackSettings {
		return this[PropertySymbol.settings];
	}

	/**
	 * Clones the track.
	 *
	 * @returns Clone.
	 */
	public clone(): MediaStreamTrack {
		const clone = new (<typeof MediaStreamTrack>this.constructor)(
			PropertySymbol.illegalConstructor
		);
		clone[PropertySymbol.kind] = this[PropertySymbol.kind];
		clone[PropertySymbol.constraints] = this[PropertySymbol.constraints];
		clone[PropertySymbol.capabilities] = this[PropertySymbol.capabilities];
		clone[PropertySymbol.settings] = this[PropertySymbol.settings];
		clone.contentHint = this.contentHint;
		clone.enabled = this.enabled;
		clone.label = this.label;
		clone.muted = this.muted;
		clone.readyState = this.readyState;
		return clone;
	}

	/**
	 * Stops the track.
	 */
	public stop(): void {
		this.readyState = 'ended';
	}

	/**
	 * Merges two objects.
	 *
	 * @param source Target.
	 * @param target Source.
	 */
	#mergeObjects(source: object, target: object): void {
		for (const key in target) {
			if (target[key] !== null && typeof target[key] === 'object' && !Array.isArray(target[key])) {
				if (typeof source[key] !== 'object') {
					source[key] = {};
				}
				this.#mergeObjects(source[key], target[key]);
			} else {
				source[key] = target[key];
			}
		}
	}
}
