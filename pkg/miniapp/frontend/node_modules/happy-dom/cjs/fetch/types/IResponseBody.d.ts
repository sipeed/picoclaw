import { ReadableStream } from 'stream/web';
import { URLSearchParams } from 'url';
import FormData from '../../form-data/FormData.cjs';
import Blob from '../../file/Blob.cjs';
type IResponseBody = ArrayBuffer | ArrayBufferView | ReadableStream | string | URLSearchParams | Blob | FormData | null;
export default IResponseBody;
//# sourceMappingURL=IResponseBody.d.ts.map