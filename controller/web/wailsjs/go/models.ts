export namespace main {
	
	export class DesktopInfo {
	    isDesktop: boolean;
	    version: string;
	    os: string;
	    arch: string;
	    port: number;
	
	    static createFrom(source: any = {}) {
	        return new DesktopInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.isDesktop = source["isDesktop"];
	        this.version = source["version"];
	        this.os = source["os"];
	        this.arch = source["arch"];
	        this.port = source["port"];
	    }
	}

}

