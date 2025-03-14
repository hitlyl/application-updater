export namespace main {
	
	export class Camera {
	    taskId: string;
	    deviceName: string;
	    url: string;
	    types: number[];
	
	    static createFrom(source: any = {}) {
	        return new Camera(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.taskId = source["taskId"];
	        this.deviceName = source["deviceName"];
	        this.url = source["url"];
	        this.types = source["types"];
	    }
	}
	export class CameraConfigResult {
	    deviceIp: string;
	    cameraName: string;
	    success: boolean;
	    message: string;
	
	    static createFrom(source: any = {}) {
	        return new CameraConfigResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deviceIp = source["deviceIp"];
	        this.cameraName = source["cameraName"];
	        this.success = source["success"];
	        this.message = source["message"];
	    }
	}
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
	export class ExcelRow {
	    deviceIp: string;
	    cameraName: string;
	    cameraInfo: string;
	
	    static createFrom(source: any = {}) {
	        return new ExcelRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deviceIp = source["deviceIp"];
	        this.cameraName = source["cameraName"];
	        this.cameraInfo = source["cameraInfo"];
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

