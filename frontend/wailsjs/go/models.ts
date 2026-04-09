export namespace clipboard {
	
	export class ClipboardEntry {
	    id: string;
	    content: string;
	    contentType: string;
	    imageData?: string;
	    imageMimeType?: string;
	    // Go type: time
	    timestamp: any;
	
	    static createFrom(source: any = {}) {
	        return new ClipboardEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.content = source["content"];
	        this.contentType = source["contentType"];
	        this.imageData = source["imageData"];
	        this.imageMimeType = source["imageMimeType"];
	        this.timestamp = this.convertValues(source["timestamp"], null);
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

export namespace peersclipsync {
	
	export class PeerClipboard {
	    peerName: string;
	    entries: clipboard.ClipboardEntry[];
	
	    static createFrom(source: any = {}) {
	        return new PeerClipboard(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.peerName = source["peerName"];
	        this.entries = this.convertValues(source["entries"], clipboard.ClipboardEntry);
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

export namespace theme {
	
	export class ThemeColors {
	    background: string;
	    foreground: string;
	    accent: string;
	    cursor: string;
	    selectionBackground: string;
	    selectionForeground: string;
	    color0: string;
	    color1: string;
	    color2: string;
	    color3: string;
	    color4: string;
	    color5: string;
	    color6: string;
	    color7: string;
	    color8: string;
	    color9: string;
	    color10: string;
	    color11: string;
	    color12: string;
	    color13: string;
	    color14: string;
	    color15: string;
	
	    static createFrom(source: any = {}) {
	        return new ThemeColors(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.background = source["background"];
	        this.foreground = source["foreground"];
	        this.accent = source["accent"];
	        this.cursor = source["cursor"];
	        this.selectionBackground = source["selectionBackground"];
	        this.selectionForeground = source["selectionForeground"];
	        this.color0 = source["color0"];
	        this.color1 = source["color1"];
	        this.color2 = source["color2"];
	        this.color3 = source["color3"];
	        this.color4 = source["color4"];
	        this.color5 = source["color5"];
	        this.color6 = source["color6"];
	        this.color7 = source["color7"];
	        this.color8 = source["color8"];
	        this.color9 = source["color9"];
	        this.color10 = source["color10"];
	        this.color11 = source["color11"];
	        this.color12 = source["color12"];
	        this.color13 = source["color13"];
	        this.color14 = source["color14"];
	        this.color15 = source["color15"];
	    }
	}

}

