export namespace cursor {
	
	export class Status {
	    path: string;
	    version: string;
	    compatible: boolean;
	    compatibleVersion: string;
	    downloadUrl: string;
	    useOpenAIKey: boolean;
	    openAIBaseUrl: string;
	    enabled: string[];
	    disabled: string[];
	    userAdded: string[];
	    cursorRunning: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.version = source["version"];
	        this.compatible = source["compatible"];
	        this.compatibleVersion = source["compatibleVersion"];
	        this.downloadUrl = source["downloadUrl"];
	        this.useOpenAIKey = source["useOpenAIKey"];
	        this.openAIBaseUrl = source["openAIBaseUrl"];
	        this.enabled = source["enabled"];
	        this.disabled = source["disabled"];
	        this.userAdded = source["userAdded"];
	        this.cursorRunning = source["cursorRunning"];
	    }
	}
	export class RestoreResult {
	    message: string;
	    status: Status;
	
	    static createFrom(source: any = {}) {
	        return new RestoreResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = source["message"];
	        this.status = this.convertValues(source["status"], Status);
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

export namespace main {
	
	export class CloseDecision {
	    action: string;
	    remember: boolean;
	
	    static createFrom(source: any = {}) {
	        return new CloseDecision(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.action = source["action"];
	        this.remember = source["remember"];
	    }
	}
	export class ProviderView {
	    id: string;
	    name: string;
	    baseUrl: string;
	    enabled: boolean;
	    catchAll: boolean;
	    hasKey: boolean;
	    online: boolean;
	    onlineErr: string;
	    models: string[];
	    lastError: string;
	
	    static createFrom(source: any = {}) {
	        return new ProviderView(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.enabled = source["enabled"];
	        this.catchAll = source["catchAll"];
	        this.hasKey = source["hasKey"];
	        this.online = source["online"];
	        this.onlineErr = source["onlineErr"];
	        this.models = source["models"];
	        this.lastError = source["lastError"];
	    }
	}
	export class Dashboard {
	    providers: ProviderView[];
	    gateway: mux.Status;
	    cursor: cursor.Status;
	    prefs: store.Prefs;
	    app: update.Status;
	
	    static createFrom(source: any = {}) {
	        return new Dashboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.providers = this.convertValues(source["providers"], ProviderView);
	        this.gateway = this.convertValues(source["gateway"], mux.Status);
	        this.cursor = this.convertValues(source["cursor"], cursor.Status);
	        this.prefs = this.convertValues(source["prefs"], store.Prefs);
	        this.app = this.convertValues(source["app"], update.Status);
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
	
	export class SaveRequest {
	    id: string;
	    name: string;
	    baseUrl: string;
	    apiKey: string;
	    enabled: boolean;
	    catchAll: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SaveRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.baseUrl = source["baseUrl"];
	        this.apiKey = source["apiKey"];
	        this.enabled = source["enabled"];
	        this.catchAll = source["catchAll"];
	    }
	}

}

export namespace mux {
	
	export class Status {
	    running: boolean;
	    addr: string;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.running = source["running"];
	        this.addr = source["addr"];
	        this.error = source["error"];
	    }
	}

}

export namespace update {
	
	export class Status {
	    current: string;
	    latest: string;
	    notes: string;
	    url: string;
	    assetName: string;
	    assetUrl: string;
	    sha256?: string;
	    available: boolean;
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new Status(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.current = source["current"];
	        this.latest = source["latest"];
	        this.notes = source["notes"];
	        this.url = source["url"];
	        this.assetName = source["assetName"];
	        this.assetUrl = source["assetUrl"];
	        this.sha256 = source["sha256"];
	        this.available = source["available"];
	        this.error = source["error"];
	    }
	}

}

export namespace store {
	
	export class Prefs {
	    theme: string;
	    closeAction: string;
	    rememberClose: boolean;
	
	    static createFrom(source: any = {}) {
	        return new Prefs(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.theme = source["theme"];
	        this.closeAction = source["closeAction"];
	        this.rememberClose = source["rememberClose"];
	    }
	}

}

