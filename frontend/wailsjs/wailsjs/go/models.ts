export namespace models {
	
	export class BackupResult {
	    ip: string;
	    success: boolean;
	    message: string;
	    backupPath: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.backupPath = source["backupPath"];
	    }
	}
	export class BackupSettings {
	    storageFolder: string;
	    regionName: string;
	    username: string;
	    password: string;
	
	    static createFrom(source: any = {}) {
	        return new BackupSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.storageFolder = source["storageFolder"];
	        this.regionName = source["regionName"];
	        this.username = source["username"];
	        this.password = source["password"];
	    }
	}
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
	    id: string;
	    ip: string;
	    buildTime: string;
	    status: string;
	    region?: string;
	
	    static createFrom(source: any = {}) {
	        return new Device(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.ip = source["ip"];
	        this.buildTime = source["buildTime"];
	        this.status = source["status"];
	        this.region = source["region"];
	    }
	}
	export class ExcelRow {
	    deviceIp: string;
	    cameraName: string;
	    cameraInfo: string;
	    deviceIndex: number;
	    selected: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ExcelRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.deviceIp = source["deviceIp"];
	        this.cameraName = source["cameraName"];
	        this.cameraInfo = source["cameraInfo"];
	        this.deviceIndex = source["deviceIndex"];
	        this.selected = source["selected"];
	    }
	}
	export class RestoreResult {
	    ip: string;
	    success: boolean;
	    message: string;
	    originalPath: string;
	    backupPath: string;
	
	    static createFrom(source: any = {}) {
	        return new RestoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.originalPath = source["originalPath"];
	        this.backupPath = source["backupPath"];
	    }
	}
	export class TimeSyncResult {
	    ip: string;
	    success: boolean;
	    message: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new TimeSyncResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ip = source["ip"];
	        this.success = source["success"];
	        this.message = source["message"];
	        this.timestamp = source["timestamp"];
	    }
	}

}

