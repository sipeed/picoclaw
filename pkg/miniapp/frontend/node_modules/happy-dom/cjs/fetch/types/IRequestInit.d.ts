import IHeadersInit from './IHeadersInit.cjs';
import AbortSignal from '../AbortSignal.cjs';
import URL from '../../url/URL.cjs';
import IRequestReferrerPolicy from './IRequestReferrerPolicy.cjs';
import IRequestRedirect from './IRequestRedirect.cjs';
import IRequestBody from './IRequestBody.cjs';
import IRequestCredentials from './IRequestCredentials.cjs';
/**
 * Fetch request init.
 */
export default interface IRequestInit {
    body?: IRequestBody;
    headers?: IHeadersInit;
    method?: string;
    redirect?: IRequestRedirect;
    signal?: AbortSignal | null;
    referrer?: '' | 'no-referrer' | 'client' | string | URL;
    credentials?: IRequestCredentials;
    referrerPolicy?: IRequestReferrerPolicy;
}
//# sourceMappingURL=IRequestInit.d.ts.map