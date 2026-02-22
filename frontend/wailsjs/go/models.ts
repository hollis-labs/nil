export namespace config {
	
	export class Vault {
	    id: string;
	    name: string;
	    path: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new Vault(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.name = source["name"];
	        this.path = source["path"];
	        this.created_at = source["created_at"];
	    }
	}

}

export namespace main {
	
	export class APIConfigResult {
	    enabled: boolean;
	    port: number;
	    api_key: string;
	
	    static createFrom(source: any = {}) {
	        return new APIConfigResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.port = source["port"];
	        this.api_key = source["api_key"];
	    }
	}
	export class FiltersResult {
	    projects: string[];
	    contexts: string[];
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new FiltersResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projects = source["projects"];
	        this.contexts = source["contexts"];
	        this.tags = source["tags"];
	    }
	}

}

export namespace store {
	
	export class Item {
	    id: number;
	    title: string;
	    priority?: string;
	    completed: boolean;
	    archived: boolean;
	    created_at: string;
	    updated_at: string;
	    due_at?: string;
	    threshold_at?: string;
	    recurrence_rule?: string;
	    source_line: string;
	    notes_md: string;
	    section: string;
	    pinned: boolean;
	    type: string;
	    inbox: boolean;
	    api_source?: string;
	    projects: string[];
	    contexts: string[];
	    tags: string[];
	
	    static createFrom(source: any = {}) {
	        return new Item(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.title = source["title"];
	        this.priority = source["priority"];
	        this.completed = source["completed"];
	        this.archived = source["archived"];
	        this.created_at = source["created_at"];
	        this.updated_at = source["updated_at"];
	        this.due_at = source["due_at"];
	        this.threshold_at = source["threshold_at"];
	        this.recurrence_rule = source["recurrence_rule"];
	        this.source_line = source["source_line"];
	        this.notes_md = source["notes_md"];
	        this.section = source["section"];
	        this.pinned = source["pinned"];
	        this.type = source["type"];
	        this.inbox = source["inbox"];
	        this.api_source = source["api_source"];
	        this.projects = source["projects"];
	        this.contexts = source["contexts"];
	        this.tags = source["tags"];
	    }
	}
	export class SearchRequest {
	    query: string;
	    projects: string[];
	    contexts: string[];
	    tags: string[];
	    statuses: string[];
	    priorities: string[];
	    date_from?: string;
	    date_to?: string;
	    page: number;
	    page_size: number;
	    sort_by: string;
	    sort_dir: string;
	    type: string;
	    include_inbox: boolean;
	
	    static createFrom(source: any = {}) {
	        return new SearchRequest(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.query = source["query"];
	        this.projects = source["projects"];
	        this.contexts = source["contexts"];
	        this.tags = source["tags"];
	        this.statuses = source["statuses"];
	        this.priorities = source["priorities"];
	        this.date_from = source["date_from"];
	        this.date_to = source["date_to"];
	        this.page = source["page"];
	        this.page_size = source["page_size"];
	        this.sort_by = source["sort_by"];
	        this.sort_dir = source["sort_dir"];
	        this.type = source["type"];
	        this.include_inbox = source["include_inbox"];
	    }
	}

}

