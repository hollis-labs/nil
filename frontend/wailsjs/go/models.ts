export namespace chat {
	
	export class ActionProposal {
	    id: number;
	    session_id: number;
	    message_id?: number;
	    action_type: string;
	    item_type: string;
	    vault_id: string;
	    payload: store.Item;
	    diff?: Record<string, any>;
	    status: string;
	    created_at: string;
	    resolved_at?: string;
	    error_msg?: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionProposal(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.session_id = source["session_id"];
	        this.message_id = source["message_id"];
	        this.action_type = source["action_type"];
	        this.item_type = source["item_type"];
	        this.vault_id = source["vault_id"];
	        this.payload = this.convertValues(source["payload"], store.Item);
	        this.diff = source["diff"];
	        this.status = source["status"];
	        this.created_at = source["created_at"];
	        this.resolved_at = source["resolved_at"];
	        this.error_msg = source["error_msg"];
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
	export class ActionResult {
	    proposal_id: number;
	    item_id?: number;
	    dry_run: boolean;
	    outcome: string;
	
	    static createFrom(source: any = {}) {
	        return new ActionResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.proposal_id = source["proposal_id"];
	        this.item_id = source["item_id"];
	        this.dry_run = source["dry_run"];
	        this.outcome = source["outcome"];
	    }
	}
	export class AuditEntry {
	    id: number;
	    proposal_id?: number;
	    action_type: string;
	    item_type: string;
	    vault_id: string;
	    item_id?: number;
	    outcome: string;
	    payload_json: string;
	    actor: string;
	    timestamp: string;
	
	    static createFrom(source: any = {}) {
	        return new AuditEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.proposal_id = source["proposal_id"];
	        this.action_type = source["action_type"];
	        this.item_type = source["item_type"];
	        this.vault_id = source["vault_id"];
	        this.item_id = source["item_id"];
	        this.outcome = source["outcome"];
	        this.payload_json = source["payload_json"];
	        this.actor = source["actor"];
	        this.timestamp = source["timestamp"];
	    }
	}
	export class ChatMessage {
	    id: number;
	    session_id: number;
	    role: string;
	    content: string;
	    created_at: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatMessage(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.session_id = source["session_id"];
	        this.role = source["role"];
	        this.content = source["content"];
	        this.created_at = source["created_at"];
	    }
	}
	export class ToolCallRecord {
	    tool_name: string;
	    input_json: string;
	    result_json: string;
	    cache_hit?: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ToolCallRecord(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.tool_name = source["tool_name"];
	        this.input_json = source["input_json"];
	        this.result_json = source["result_json"];
	        this.cache_hit = source["cache_hit"];
	    }
	}
	export class ChatResponse {
	    message: ChatMessage;
	    proposal_id?: number;
	    proposal?: ActionProposal;
	    tool_calls?: ToolCallRecord[];
	    error?: string;
	
	    static createFrom(source: any = {}) {
	        return new ChatResponse(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.message = this.convertValues(source["message"], ChatMessage);
	        this.proposal_id = source["proposal_id"];
	        this.proposal = this.convertValues(source["proposal"], ActionProposal);
	        this.tool_calls = this.convertValues(source["tool_calls"], ToolCallRecord);
	        this.error = source["error"];
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
	export class ChatSession {
	    id: number;
	    vault_id: string;
	    started_at: string;
	    ended_at?: string;
	    dry_run: boolean;
	
	    static createFrom(source: any = {}) {
	        return new ChatSession(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.id = source["id"];
	        this.vault_id = source["vault_id"];
	        this.started_at = source["started_at"];
	        this.ended_at = source["ended_at"];
	        this.dry_run = source["dry_run"];
	    }
	}

}

export namespace config {
	
	export class VaultCap {
	    read: boolean;
	    write: boolean;
	    delete: boolean;
	    directCreate: boolean;
	
	    static createFrom(source: any = {}) {
	        return new VaultCap(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.read = source["read"];
	        this.write = source["write"];
	        this.delete = source["delete"];
	        this.directCreate = source["directCreate"];
	    }
	}
	export class ChatConfig {
	    enabled: boolean;
	    apiKey: string;
	    model: string;
	    dryRun: boolean;
	    vaultCaps: Record<string, VaultCap>;
	
	    static createFrom(source: any = {}) {
	        return new ChatConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.enabled = source["enabled"];
	        this.apiKey = source["apiKey"];
	        this.model = source["model"];
	        this.dryRun = source["dryRun"];
	        this.vaultCaps = this.convertValues(source["vaultCaps"], VaultCap, true);
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
	    notes_text?: string;
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
	        this.notes_text = source["notes_text"];
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

