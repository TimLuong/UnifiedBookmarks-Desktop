export namespace browser {
	
	export class Bookmark {
	    id: string;
	    title: string;
	    url: string;
	    folderPath: string;
	    rootSection: string;
	    browser: string;
	    profileDir: string;
	    displayName: string;
	    dateAdded: string;
	    category: string;
	    confidence: number;
	    paraType: string;
	    paraContext: string;
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new Bookmark(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.url = source["url"];
	        this.folderPath = source["folderPath"];
	        this.rootSection = source["rootSection"];
	        this.browser = source["browser"];
	        this.profileDir = source["profileDir"];
	        this.displayName = source["displayName"];
	        this.dateAdded = source["dateAdded"];
	        this.category = source["category"];
	        this.confidence = source["confidence"];
	        this.paraType = source["paraType"];
	        this.paraContext = source["paraContext"];
	        this.tags = source["tags"];
	    }
	}
	export class Profile {
	    browser: string;
	    browserLabel: string;
	    profileDir: string;
	    displayName: string;
	    bookmarksPath: string;
	    hasBookmarks: boolean;
	    userDataDir: string;
	
	    static createFrom(source: any = {}) {
	        return new Profile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browser = source["browser"];
	        this.browserLabel = source["browserLabel"];
	        this.profileDir = source["profileDir"];
	        this.displayName = source["displayName"];
	        this.bookmarksPath = source["bookmarksPath"];
	        this.hasBookmarks = source["hasBookmarks"];
	        this.userDataDir = source["userDataDir"];
	    }
	}

}

export namespace engine {
	
	export class CategorizeResult {
	    bookmarks: browser.Bookmark[];
	    totalTokens: number;
	    totalBatches: number;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new CategorizeResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bookmarks = this.convertValues(source["bookmarks"], browser.Bookmark);
	        this.totalTokens = source["totalTokens"];
	        this.totalBatches = source["totalBatches"];
	        this.model = source["model"];
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
	
	export class AIConfig {
	    endpoint: string;
	    apiKey: string;
	    model: string;
	    systemPrompt: string;
	
	    static createFrom(source: any = {}) {
	        return new AIConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.endpoint = source["endpoint"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.systemPrompt = source["systemPrompt"];
	    }
	}
	export class AnalysisCacheMeta {
	    exists: boolean;
	    timestamp: string;
	    count: number;
	    model: string;
	    tokens: number;
	
	    static createFrom(source: any = {}) {
	        return new AnalysisCacheMeta(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.exists = source["exists"];
	        this.timestamp = source["timestamp"];
	        this.count = source["count"];
	        this.model = source["model"];
	        this.tokens = source["tokens"];
	    }
	}
	export class AppConfig {
	    model: string;
	    batchSize: number;
	    hasApiKey: boolean;
	    backupDir: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.model = source["model"];
	        this.batchSize = source["batchSize"];
	        this.hasApiKey = source["hasApiKey"];
	        this.backupDir = source["backupDir"];
	    }
	}
	export class BookmarkDiffRow {
	    title: string;
	    url: string;
	    domain: string;
	    folderPath: string;
	    category: string;
	    status: string;
	
	    static createFrom(source: any = {}) {
	        return new BookmarkDiffRow(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.title = source["title"];
	        this.url = source["url"];
	        this.domain = source["domain"];
	        this.folderPath = source["folderPath"];
	        this.category = source["category"];
	        this.status = source["status"];
	    }
	}
	export class CollectResult {
	    totalRaw: number;
	    totalDeduped: number;
	    duplicatesUrl: number;
	    duplicatesFuzzy: number;
	
	    static createFrom(source: any = {}) {
	        return new CollectResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.totalRaw = source["totalRaw"];
	        this.totalDeduped = source["totalDeduped"];
	        this.duplicatesUrl = source["duplicatesUrl"];
	        this.duplicatesFuzzy = source["duplicatesFuzzy"];
	    }
	}
	export class FetchPageResult {
	    html: string;
	    error: string;
	
	    static createFrom(source: any = {}) {
	        return new FetchPageResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.html = source["html"];
	        this.error = source["error"];
	    }
	}
	export class FolderNode {
	    name: string;
	    path: string;
	    count: number;
	    urls: string[];
	    children: FolderNode[];
	
	    static createFrom(source: any = {}) {
	        return new FolderNode(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.name = source["name"];
	        this.path = source["path"];
	        this.count = source["count"];
	        this.urls = source["urls"];
	        this.children = this.convertValues(source["children"], FolderNode);
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
	export class FolderSettings {
	    maxDepth: number;
	    minFolderItems: number;
	    maxFolderItems: number;
	    smartRenamePrefix: boolean;
	    sortAlphaInFolder: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FolderSettings(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.maxDepth = source["maxDepth"];
	        this.minFolderItems = source["minFolderItems"];
	        this.maxFolderItems = source["maxFolderItems"];
	        this.smartRenamePrefix = source["smartRenamePrefix"];
	        this.sortAlphaInFolder = source["sortAlphaInFolder"];
	    }
	}
	export class LoadLastAnalysisResult {
	    bookmarks: browser.Bookmark[];
	    profiles: browser.Profile[];
	    model: string;
	    totalTokens: number;
	
	    static createFrom(source: any = {}) {
	        return new LoadLastAnalysisResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.bookmarks = this.convertValues(source["bookmarks"], browser.Bookmark);
	        this.profiles = this.convertValues(source["profiles"], browser.Profile);
	        this.model = source["model"];
	        this.totalTokens = source["totalTokens"];
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
	export class ProfileDiff {
	    browser: string;
	    browserLabel: string;
	    profileDir: string;
	    displayName: string;
	    beforeCount: number;
	    afterCount: number;
	    added: number;
	    removed: number;
	    unchanged: number;
	    addedSample: browser.Bookmark[];
	    removedSample: browser.Bookmark[];
	
	    static createFrom(source: any = {}) {
	        return new ProfileDiff(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browser = source["browser"];
	        this.browserLabel = source["browserLabel"];
	        this.profileDir = source["profileDir"];
	        this.displayName = source["displayName"];
	        this.beforeCount = source["beforeCount"];
	        this.afterCount = source["afterCount"];
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.unchanged = source["unchanged"];
	        this.addedSample = this.convertValues(source["addedSample"], browser.Bookmark);
	        this.removedSample = this.convertValues(source["removedSample"], browser.Bookmark);
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
	export class ProfileDiffDetail {
	    browser: string;
	    browserLabel: string;
	    profileDir: string;
	    displayName: string;
	    before: BookmarkDiffRow[];
	    after: BookmarkDiffRow[];
	    added: number;
	    removed: number;
	    unchanged: number;
	
	    static createFrom(source: any = {}) {
	        return new ProfileDiffDetail(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browser = source["browser"];
	        this.browserLabel = source["browserLabel"];
	        this.profileDir = source["profileDir"];
	        this.displayName = source["displayName"];
	        this.before = this.convertValues(source["before"], BookmarkDiffRow);
	        this.after = this.convertValues(source["after"], BookmarkDiffRow);
	        this.added = source["added"];
	        this.removed = source["removed"];
	        this.unchanged = source["unchanged"];
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
	export class ProfileTree {
	    browser: string;
	    browserLabel: string;
	    profileDir: string;
	    displayName: string;
	    totalCount: number;
	    roots: FolderNode[];
	
	    static createFrom(source: any = {}) {
	        return new ProfileTree(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browser = source["browser"];
	        this.browserLabel = source["browserLabel"];
	        this.profileDir = source["profileDir"];
	        this.displayName = source["displayName"];
	        this.totalCount = source["totalCount"];
	        this.roots = this.convertValues(source["roots"], FolderNode);
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
	export class PromptProfile {
	    id: string;
	    name: string;
	    description: string;
	    content: string;
	    isBuiltin: boolean;
	
	    static createFrom(source: any = {}) {
	        return new PromptProfile(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.description = source["description"];
	        this.content = source["content"];
	        this.isBuiltin = source["isBuiltin"];
	    }
	}
	export class ScanResult {
	    profiles: browser.Profile[];
	    os: string;
	
	    static createFrom(source: any = {}) {
	        return new ScanResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.profiles = this.convertValues(source["profiles"], browser.Profile);
	        this.os = source["os"];
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
	export class SyncPreviewResult {
	    diffs: ProfileDiff[];
	    totalBefore: number;
	    totalAfter: number;
	
	    static createFrom(source: any = {}) {
	        return new SyncPreviewResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.diffs = this.convertValues(source["diffs"], ProfileDiff);
	        this.totalBefore = source["totalBefore"];
	        this.totalAfter = source["totalAfter"];
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
	export class SyncResponse {
	    results: sync.WriteResult[];
	    backups: sync.Snapshot[];
	
	    static createFrom(source: any = {}) {
	        return new SyncResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.results = this.convertValues(source["results"], sync.WriteResult);
	        this.backups = this.convertValues(source["backups"], sync.Snapshot);
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
	export class TestResult {
	    ok: boolean;
	    message: string;
	    model: string;
	
	    static createFrom(source: any = {}) {
	        return new TestResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ok = source["ok"];
	        this.message = source["message"];
	        this.model = source["model"];
	    }
	}

}

export namespace sync {
	
	export class Snapshot {
	    id: string;
	    timestamp: string;
	    browser: string;
	    profile: string;
	    count: number;
	    filePath: string;
	    sizeBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new Snapshot(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.timestamp = source["timestamp"];
	        this.browser = source["browser"];
	        this.profile = source["profile"];
	        this.count = source["count"];
	        this.filePath = source["filePath"];
	        this.sizeBytes = source["sizeBytes"];
	    }
	}
	export class WriteResult {
	    browser: string;
	    profile: string;
	    status: string;
	    reason?: string;
	    written: number;
	    folders: number;
	
	    static createFrom(source: any = {}) {
	        return new WriteResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.browser = source["browser"];
	        this.profile = source["profile"];
	        this.status = source["status"];
	        this.reason = source["reason"];
	        this.written = source["written"];
	        this.folders = source["folders"];
	    }
	}

}

