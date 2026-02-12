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

