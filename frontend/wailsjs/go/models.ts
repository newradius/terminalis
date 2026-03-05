export namespace config {
	
	export class AppConfig {
	    fontSize: number;
	    theme: string;
	    defaultPort: number;
	    defaultUsername: string;
	    connectTimeout: number;
	    scrollbackLines: number;
	    terminalBackground: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.fontSize = source["fontSize"];
	        this.theme = source["theme"];
	        this.defaultPort = source["defaultPort"];
	        this.defaultUsername = source["defaultUsername"];
	        this.connectTimeout = source["connectTimeout"];
	        this.scrollbackLines = source["scrollbackLines"];
	        this.terminalBackground = source["terminalBackground"];
	    }
	}

}

export namespace main {
	
	export class ConnectRequest {
	    tabId: string;
	    sessionId: string;
	    password?: string;
	    cols: number;
	    rows: number;
	
	    static createFrom(source: any = {}) {
	        return new ConnectRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tabId = source["tabId"];
	        this.sessionId = source["sessionId"];
	        this.password = source["password"];
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	    }
	}
	export class OpenShellRequest {
	    tabId: string;
	    shell: string;
	    cols: number;
	    rows: number;
	
	    static createFrom(source: any = {}) {
	        return new OpenShellRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tabId = source["tabId"];
	        this.shell = source["shell"];
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	    }
	}
	export class QuickConnectRequest {
	    tabId: string;
	    connString: string;
	    password?: string;
	    cols: number;
	    rows: number;
	
	    static createFrom(source: any = {}) {
	        return new QuickConnectRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tabId = source["tabId"];
	        this.connString = source["connString"];
	        this.password = source["password"];
	        this.cols = source["cols"];
	        this.rows = source["rows"];
	    }
	}

}

export namespace models {
	
	export class Folder {
	    id: string;
	    name: string;
	    parentId?: string;
	    color?: string;
	    expanded: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Folder(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.parentId = source["parentId"];
	        this.color = source["color"];
	        this.expanded = source["expanded"];
	    }
	}
	export class Session {
	    id: string;
	    name: string;
	    host: string;
	    port: number;
	    username: string;
	    authMethod: string;
	    password?: string;
	    privateKeyPath?: string;
	    passphrase?: string;
	    folderId: string;
	    color?: string;
	    compression: boolean;
	    keepAlive: number;
	    terminalType?: string;
	    systemTerminal?: string;
	    createdAt: number;
	    updatedAt: number;
	
	    static createFrom(source: any = {}) {
	        return new Session(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.host = source["host"];
	        this.port = source["port"];
	        this.username = source["username"];
	        this.authMethod = source["authMethod"];
	        this.password = source["password"];
	        this.privateKeyPath = source["privateKeyPath"];
	        this.passphrase = source["passphrase"];
	        this.folderId = source["folderId"];
	        this.color = source["color"];
	        this.compression = source["compression"];
	        this.keepAlive = source["keepAlive"];
	        this.terminalType = source["terminalType"];
	        this.systemTerminal = source["systemTerminal"];
	        this.createdAt = source["createdAt"];
	        this.updatedAt = source["updatedAt"];
	    }
	}
	export class TreeNode {
	    id: string;
	    name: string;
	    type: string;
	    color?: string;
	    expanded?: boolean;
	    children?: TreeNode[];
	    session?: Session;
	
	    static createFrom(source: any = {}) {
	        return new TreeNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.type = source["type"];
	        this.color = source["color"];
	        this.expanded = source["expanded"];
	        this.children = this.convertValues(source["children"], TreeNode);
	        this.session = this.convertValues(source["session"], Session);
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

}

export namespace terminal {
	
	export class ShellInfo {
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new ShellInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}
	export class TerminalInfo {
	    name: string;
	    path: string;
	
	    static createFrom(source: any = {}) {
	        return new TerminalInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	    }
	}

}

