export namespace main {
	
	export class Device {
	    ip: string;
	    buildTime: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.buildTime = source["buildTime"];
	        this.status = source["status"];
	    }
	}
	export class UpdateResult {
	    ip: string;
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new UpdateResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}

}

