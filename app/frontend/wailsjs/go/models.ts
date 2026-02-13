export namespace models {
	
	export class ConnectionProfile {
	    id: string;
	    name: string;
	    protocol: string;
	    host: string;
	    port: number;
	    authType: string;
	    credentials?: Record<string, string>;
	    path: string;
	    metadata?: Record<string, any>;
	    username?: string;
	    credentialsMasked?: Record<string, boolean>;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new ConnectionProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.protocol = source["protocol"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.authType = source["authType"];
	        this.credentials = source["credentials"];
	        this.path = source["path"];
	        this.metadata = source["metadata"];
	        this.username = source["username"];
	        this.credentialsMasked = source["credentialsMasked"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class FileEntry {
	    name: string;
	    path: string;
	    isDir: boolean;
	    size: number;
	    modifiedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new FileEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.isDir = source["isDir"];
	        this.size = source["size"];
	        this.modifiedAt = source["modifiedAt"];
	    }
	}
	export class ListFilesResult {
	    path: string;
	    entries: FileEntry[];
	
	    static createFrom(source: any = {}) {
	        return new ListFilesResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.entries = this.convertValues(source["entries"], FileEntry);
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
	export class TransferItem {
	    id: string;
	    sessionID: string;
	    protocol: string;
	    direction: string;
	    localPath: string;
	    remotePath: string;
	    bytesTotal: number;
	    bytesTransferred: number;
	    startedAt: number;
	    finishedAt: number;
	    status: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new TransferItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.sessionID = source["sessionID"];
	        this.protocol = source["protocol"];
	        this.direction = source["direction"];
	        this.localPath = source["localPath"];
	        this.remotePath = source["remotePath"];
	        this.bytesTotal = source["bytesTotal"];
	        this.bytesTransferred = source["bytesTransferred"];
	        this.startedAt = source["startedAt"];
	        this.finishedAt = source["finishedAt"];
	        this.status = source["status"];
	        this.error = source["error"];
	    }
	}

}

export namespace services {
	
	export class MasterPasswordStatus {
	    unlocked: boolean;
	    hasEncryptedStore: boolean;
	
	    static createFrom(source: any = {}) {
	        return new MasterPasswordStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.unlocked = source["unlocked"];
	        this.hasEncryptedStore = source["hasEncryptedStore"];
	    }
	}

}

