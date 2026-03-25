import Headers from '../../Headers.js';

export default interface ICachableResponse {
	status: number;
	statusText: string;
	url: string;
	headers: Headers;
	body: Buffer | null;
	waitingForBody: boolean;
}
