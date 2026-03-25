import Headers from '../../Headers.js';

export default interface ICachablePreflightRequest {
	url: string;
	method: string;
	headers: Headers;
}
