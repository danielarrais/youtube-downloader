export namespace main {
	
	export class Config {
	    download_dir: string;
	    quality: string;
	    language: string;
	
	    static createFrom(source: any = {}) {
	        return new Config(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.download_dir = source["download_dir"];
	        this.quality = source["quality"];
	        this.language = source["language"];
	    }
	}
	export class DownloadProgress {
	    percent: number;
	    downloaded_bytes: number;
	    total_bytes: number;
	    speed: string;
	    eta: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadProgress(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.percent = source["percent"];
	        this.downloaded_bytes = source["downloaded_bytes"];
	        this.total_bytes = source["total_bytes"];
	        this.speed = source["speed"];
	        this.eta = source["eta"];
	    }
	}
	export class DownloadItem {
	    id: string;
	    url: string;
	    title: string;
	    quality: string;
	    status: string;
	    progress: DownloadProgress;
	    error?: string;
	    file_path?: string;
	    file_size?: number;
	    thumbnail_url?: string;
	    created_at: string;
	    started_at?: string;
	    completed_at?: string;
	
	    static createFrom(source: any = {}) {
	        return new DownloadItem(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.title = source["title"];
	        this.quality = source["quality"];
	        this.status = source["status"];
	        this.progress = this.convertValues(source["progress"], DownloadProgress);
	        this.error = source["error"];
	        this.file_path = source["file_path"];
	        this.file_size = source["file_size"];
	        this.thumbnail_url = source["thumbnail_url"];
	        this.created_at = source["created_at"];
	        this.started_at = source["started_at"];
	        this.completed_at = source["completed_at"];
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
	
	export class PlaylistVideo {
	    id: string;
	    url: string;
	    title: string;
	    author: string;
	    duration_seconds: number;
	    thumbnail_url: string;
	    available: boolean;
	    unavailable_reason?: string;
	    index: number;
	
	    static createFrom(source: any = {}) {
	        return new PlaylistVideo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.url = source["url"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.duration_seconds = source["duration_seconds"];
	        this.thumbnail_url = source["thumbnail_url"];
	        this.available = source["available"];
	        this.unavailable_reason = source["unavailable_reason"];
	        this.index = source["index"];
	    }
	}
	export class PlaylistInfo {
	    id: string;
	    title: string;
	    author: string;
	    videos: PlaylistVideo[];
	
	    static createFrom(source: any = {}) {
	        return new PlaylistInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.author = source["author"];
	        this.videos = this.convertValues(source["videos"], PlaylistVideo);
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
	
	export class QueueStats {
	    total: number;
	    pending: number;
	    downloading: number;
	    completed: number;
	    failed: number;
	    paused: boolean;
	
	    static createFrom(source: any = {}) {
	        return new QueueStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.pending = source["pending"];
	        this.downloading = source["downloading"];
	        this.completed = source["completed"];
	        this.failed = source["failed"];
	        this.paused = source["paused"];
	    }
	}

}

