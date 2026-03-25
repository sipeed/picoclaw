type IRequestReferrerPolicy =
	| ''
	| 'no-referrer'
	| 'no-referrer-when-downgrade'
	| 'same-origin'
	| 'origin'
	| 'strict-origin'
	| 'origin-when-cross-origin'
	| 'strict-origin-when-cross-origin'
	| 'unsafe-url';
export default IRequestReferrerPolicy;
