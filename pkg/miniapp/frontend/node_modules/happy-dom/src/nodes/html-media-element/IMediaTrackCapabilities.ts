export default interface IMediaTrackCapabilities {
	aspectRatio: {
		max: number;
		min: number;
	};
	deviceId: string;
	facingMode: [];
	frameRate: {
		max: number;
		min: number;
	};
	height: {
		max: number;
		min: number;
	};
	resizeMode: string[];
	width: {
		max: number;
		min: number;
	};
}
