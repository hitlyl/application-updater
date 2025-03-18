export namespace main {
	
	export class ExtraDef {
	    Name: string;
	    Desc: string;
	    Type: string;
	    Unit: string;
	    Default: string;
	
	    static createFrom(source: any = {}) {
	        return new ExtraDef(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Name = source["Name"];
	        this.Desc = source["Desc"];
	        this.Type = source["Type"];
	        this.Unit = source["Unit"];
	        this.Default = source["Default"];
	    }
	}
	export class ExtraConfig {
	    camera_index: string;
	    defs: ExtraDef[];
	
	    static createFrom(source: any = {}) {
	        return new ExtraConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.camera_index = source["camera_index"];
	        this.defs = this.convertValues(source["defs"], ExtraDef);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class HotAreaPoint {
	    X: number;
	    Y: number;
	
	    static createFrom(source: any = {}) {
	        return new HotAreaPoint(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.X = source["X"];
	        this.Y = source["Y"];
	    }
	}
	export class DetectInfo {
	    Id: number;
	    HotArea: HotAreaPoint[];
	
	    static createFrom(source: any = {}) {
	        return new DetectInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Id = source["Id"];
	        this.HotArea = this.convertValues(source["HotArea"], HotAreaPoint);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	export class TargetSize {
	    MinDetect: number;
	    MaxDetect: number;
	
	    static createFrom(source: any = {}) {
	        return new TargetSize(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.MinDetect = source["MinDetect"];
	        this.MaxDetect = source["MaxDetect"];
	    }
	}
	export class Algorithm {
	    Type: number;
	    TrackInterval: number;
	    DetectInterval: number;
	    AlarmInterval: number;
	    threshold: number;
	    TargetSize: TargetSize;
	    DetectInfos: DetectInfo[];
	    TripWire: any;
	    ExtraConfig: ExtraConfig;
	
	    static createFrom(source: any = {}) {
	        return new Algorithm(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.Type = source["Type"];
	        this.TrackInterval = source["TrackInterval"];
	        this.DetectInterval = source["DetectInterval"];
	        this.AlarmInterval = source["AlarmInterval"];
	        this.threshold = source["threshold"];
	        this.TargetSize = this.convertValues(source["TargetSize"], TargetSize);
	        this.DetectInfos = this.convertValues(source["DetectInfos"], DetectInfo);
	        this.TripWire = source["TripWire"];
	        this.ExtraConfig = this.convertValues(source["ExtraConfig"], ExtraConfig);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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
	export class DeviceInfo {
	    codeName: string;
	    name: string;
	    resolution: string;
	    url: string;
	    width: number;
	    height: number;
	
	    static createFrom(source: any = {}) {
	        return new DeviceInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.codeName = source["codeName"];
	        this.name = source["name"];
	        this.resolution = source["resolution"];
	        this.url = source["url"];
	        this.width = source["width"];
	        this.height = source["height"];
	    }
	}
	export class CameraConfig {
	    device: DeviceInfo;
	    algorithms: Algorithm[];
	
	    static createFrom(source: any = {}) {
	        return new CameraConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.device = this.convertValues(source["device"], DeviceInfo);
	        this.algorithms = this.convertValues(source["algorithms"], Algorithm);
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
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

