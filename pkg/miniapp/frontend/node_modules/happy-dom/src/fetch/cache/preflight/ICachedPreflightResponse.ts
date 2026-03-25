export default interface ICachedPreflightResponse {
	allowOrigin: string;
	allowMethods: string[];
	expires: number;
}
