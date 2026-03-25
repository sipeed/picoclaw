import { URLSearchParams } from 'url';
import FormData from '../../form-data/FormData.js';
import Blob from '../../file/Blob.js';
import { ReadableStream } from 'stream/web';

type IRequestBody =
	| ArrayBuffer
	| ArrayBufferView
	| ReadableStream
	| string
	| URLSearchParams
	| Blob
	| FormData
	| null;

export default IRequestBody;
