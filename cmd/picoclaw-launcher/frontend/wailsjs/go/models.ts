export namespace main {
	
	export class ChatResult {
	    success: boolean;
	    response: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.response = source["response"];
	        this.error = source["error"];
	    }
	}
	export class GatewayStatus {
	    status: string;
	    model: string;
	    logs: string[];
	    total: number;
	
	    static createFrom(source: any = {}) {
	        return new GatewayStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.status = source["status"];
	        this.model = source["model"];
	        this.logs = source["logs"];
	        this.total = source["total"];
	    }
	}
	export class SaveSetupResult {
	    success: boolean;
	    config_path: string;
	    workspace: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new SaveSetupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.config_path = source["config_path"];
	        this.workspace = source["workspace"];
	        this.error = source["error"];
	    }
	}
	export class SetupStatusResult {
	    needs_setup: boolean;
	    config_path: string;
	
	    static createFrom(source: any = {}) {
	        return new SetupStatusResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.needs_setup = source["needs_setup"];
	        this.config_path = source["config_path"];
	    }
	}
	export class TestLLMRequest {
	    api_key: string;
	    api_base: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new TestLLMRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.api_key = source["api_key"];
	        this.api_base = source["api_base"];
	        this.model = source["model"];
	    }
	}
	export class TestLLMResult {
	    success: boolean;
	    response: string;
	    model: string;
	    protocol: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new TestLLMResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.success = source["success"];
	        this.response = source["response"];
	        this.model = source["model"];
	        this.protocol = source["protocol"];
	        this.error = source["error"];
	    }
	}

}

