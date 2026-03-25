import Headers from '../../Headers.cjs';
export default interface ICachableResponse {
    status: number;
    statusText: string;
    url: string;
    headers: Headers;
    body: Buffer | null;
    waitingForBody: boolean;
}
//# sourceMappingURL=ICachableResponse.d.ts.map